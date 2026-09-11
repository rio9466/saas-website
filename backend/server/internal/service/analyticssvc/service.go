// Package analyticssvc implements public page-view ingest and the administrator
// analytics overview. It re-checks RBAC from primary storage on reads.
package analyticssvc

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/analytics"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	"github.com/rio9466/easy-admin/server/internal/platform/ratelimit"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
)

// Page-view ingest rate limit: a fixed window per source IP. The limit is
// intentionally loose (it is only statistics); a Redis failure fails open and
// accepts the report.
const (
	pageViewRateWindow = time.Hour
	pageViewRateLimit  = 1200
	pageViewRatePrefix = "easy-admin:rl:page-view:"
)

// Actor is the request-scoped administrator identity for analytics reads.
type Actor struct {
	AdminID          int64
	AdminUsername    string
	AdminDisplayName string
	AdminRoleCodes   []string
	RequestID        string
	SourceIP         string
	UserAgent        string
}

// Service records public page views and serves administrator analytics
// aggregations.
type Service struct {
	views   *primary.PageViewRepository
	admins  *primary.AdminRepository
	limiter *ratelimit.Limiter
	logger  *slog.Logger
}

// New constructs the analytics service. All collaborators are required.
func New(views *primary.PageViewRepository, admins *primary.AdminRepository, limiter *ratelimit.Limiter, logger *slog.Logger) (*Service, error) {
	if views == nil {
		return nil, errors.New("page view repository is required")
	}
	if admins == nil {
		return nil, errors.New("admin repository is required")
	}
	if limiter == nil {
		return nil, errors.New("rate limiter is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{views: views, admins: admins, limiter: limiter, logger: logger}, nil
}

// RecordPageView validates and persists one public page-view report. An invalid
// path is silently dropped (nil error, nothing stored). When the per-IP limit
// is exceeded it returns a rate-limited AppError; a Redis failure fails open and
// still stores the report.
func (s *Service) RecordPageView(ctx context.Context, in analytics.PageViewInput, sourceIP string) error {
	if !analytics.ValidPath(in.Path) {
		return nil
	}
	if !s.allowRate(ctx, sourceIP) {
		return apperr.New(http.StatusTooManyRequests, apperr.CodeRateLimited, "rate limited", apperr.ErrRateLimited)
	}
	view := &analytics.PageView{
		Path:   in.Path,
		Source: analytics.SourceFromReferrer(in.Referrer),
		Locale: analytics.NormalizeLocale(in.Locale),
	}
	if err := s.views.CreatePageView(ctx, view); err != nil {
		return mapError(err)
	}
	return nil
}

// Overview returns the aggregated page views and top sources for the requested
// range, re-checking the read permission from primary storage. An unknown range
// is a validation error.
func (s *Service) Overview(ctx context.Context, actor Actor, rangeName string) (*analytics.Overview, error) {
	if err := s.requirePermission(ctx, actor, analytics.PermissionRead); err != nil {
		return nil, err
	}
	now := time.Now()
	window, err := analytics.ResolveWindow(rangeName, now)
	if err != nil {
		return nil, validationError("range must be today, 7d, or 30d")
	}
	pv, sourceCount, sources, err := s.views.OverviewWindow(ctx, window.Start, now, analytics.SourcesLimit)
	if err != nil {
		return nil, mapError(err)
	}
	return &analytics.Overview{Range: window.Name, PV: pv, SourceCount: sourceCount, Sources: sources}, nil
}

// allowRate reports whether sourceIP is within the hourly limit. Redis errors
// fail open (accept) because analytics ingest is best-effort; an exceeded limit
// returns false.
func (s *Service) allowRate(ctx context.Context, sourceIP string) bool {
	key := pageViewRatePrefix + strings.TrimSpace(sourceIP)
	ok, err := s.limiter.Allow(ctx, key, pageViewRateLimit, pageViewRateWindow)
	if err != nil {
		s.logger.Warn("page view rate limiter unavailable; accepting report", "error", err)
		return true
	}
	return ok
}

// requirePermission re-checks RBAC from primary storage; transport-level
// permission checks are never the only authorization.
func (s *Service) requirePermission(ctx context.Context, actor Actor, code string) error {
	if actor.AdminID <= 0 {
		return apperr.New(http.StatusUnauthorized, apperr.CodeUnauthorized, "unauthorized", apperr.ErrUnauthorized)
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return apperr.New(http.StatusForbidden, apperr.CodeForbidden, "forbidden", apperr.ErrForbidden)
	}
	hasSuper, err := s.admins.AdministratorHasRole(ctx, actor.AdminID, adminauth.RoleSuperAdmin)
	if err != nil {
		return mapError(err)
	}
	if hasSuper {
		return nil
	}
	perms, err := s.admins.EffectivePermissionCodes(ctx, actor.AdminID)
	if err != nil {
		return mapError(err)
	}
	for _, p := range perms {
		if p == code {
			return nil
		}
	}
	return apperr.New(http.StatusForbidden, apperr.CodeForbidden, "forbidden", apperr.ErrForbidden)
}
