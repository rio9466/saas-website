package contentsvc

import (
	"context"
	"strings"
	"time"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	"github.com/rio9466/easy-admin/server/internal/domain/contact"
)

// Public contact-submission rate limit: a fixed window of five submissions per
// hour per source IP. Redis failures fail closed (reject) through allowRate.
const (
	contactRateWindow = time.Hour
	contactRateLimit  = 5
	contactRatePrefix = "easy-admin:rl:contact:"
)

// SubmitContact validates and persists one public contact submission, then
// best-effort sends a notification email. The honeypot returns a silent
// success without touching the rate limiter or primary storage.
func (s *Service) SubmitContact(ctx context.Context, in contact.SubmissionInput, sourceIP, userAgent string) (*contact.Receipt, error) {
	if in.HoneypotTripped() {
		return &contact.Receipt{SubmittedAt: time.Now().UTC()}, nil
	}
	normalized := in.Normalized()
	if err := contact.Validate(normalized); err != nil {
		return nil, validationError(err.Error())
	}
	if err := s.allowContactRate(ctx, sourceIP); err != nil {
		return nil, err
	}

	submission := &contact.Submission{
		Name:      normalized.Name,
		Email:     normalized.Email,
		Company:   normalized.Company,
		Message:   normalized.Message,
		Locale:    normalized.Locale,
		Status:    contact.StatusNew,
		SourceIP:  strings.TrimSpace(sourceIP),
		UserAgent: strings.TrimSpace(userAgent),
	}
	if err := s.inbox.CreateContactSubmission(ctx, submission); err != nil {
		return nil, mapError(err)
	}

	s.notifyContactSubmission(ctx, submission)
	return &contact.Receipt{ID: submission.ID, SubmittedAt: submission.CreatedAt}, nil
}

// allowContactRate enforces the per-IP fixed window. Redis errors fail closed.
func (s *Service) allowContactRate(ctx context.Context, sourceIP string) error {
	key := contactRatePrefix + strings.TrimSpace(sourceIP)
	ok, err := s.limiter.Allow(ctx, key, contactRateLimit, contactRateWindow)
	if err != nil || !ok {
		return apperr.New(429, apperr.CodeRateLimited, "rate limited", apperr.ErrRateLimited)
	}
	return nil
}

// ListContactSubmissions returns a paginated inbox page, newest first,
// optionally filtered by status.
func (s *Service) ListContactSubmissions(ctx context.Context, actor Actor, status string, page, pageSize int) (adminauth.Page[contact.Submission], error) {
	if err := s.requirePermission(ctx, actor, contact.PermissionRead); err != nil {
		return adminauth.Page[contact.Submission]{}, err
	}
	filter := strings.TrimSpace(status)
	if filter != "" && !contact.ValidStatus(filter) {
		return adminauth.Page[contact.Submission]{}, validationError("status must be new, read, or handled")
	}
	items, total, err := s.inbox.ListContactSubmissions(ctx, filter, page, pageSize)
	if err != nil {
		return adminauth.Page[contact.Submission]{}, mapError(err)
	}
	return adminauth.Page[contact.Submission]{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// GetContactSubmission returns one submission.
func (s *Service) GetContactSubmission(ctx context.Context, actor Actor, id int64) (*contact.Submission, error) {
	if err := s.requirePermission(ctx, actor, contact.PermissionRead); err != nil {
		return nil, err
	}
	submission, err := s.inbox.GetContactSubmission(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	return submission, nil
}

// UpdateContactSubmissionStatus changes only the status of one submission and
// records a pending-first audit event.
func (s *Service) UpdateContactSubmissionStatus(ctx context.Context, actor Actor, id int64, status string) (*contact.Submission, error) {
	if err := s.requirePermission(ctx, actor, contact.PermissionManage); err != nil {
		return nil, err
	}
	next := strings.TrimSpace(status)
	if !contact.ValidStatus(next) {
		return nil, validationError("status must be new, read, or handled")
	}
	if _, err := s.inbox.GetContactSubmission(ctx, id); err != nil {
		return nil, mapError(err)
	}
	details := map[string]any{"status": next}
	if err := s.withPendingAudit(ctx, actor, ActionContactStatusUpdate, ResourceContactSubmission, idString(id), details, func() error {
		return s.inbox.UpdateContactSubmissionStatus(ctx, id, next)
	}); err != nil {
		return nil, err
	}
	return s.inbox.GetContactSubmission(ctx, id)
}
