package contentsvc

import (
	"context"
	"fmt"
	"time"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
)

// Audit action identifiers for content events.
const (
	ActionSiteSettingsUpdate = "content.site_settings.update"

	ActionNavigationCreate = "content.navigation.create"
	ActionNavigationUpdate = "content.navigation.update"
	ActionNavigationDelete = "content.navigation.delete"

	ActionHomeSectionCreate = "content.home_section.create"
	ActionHomeSectionUpdate = "content.home_section.update"
	ActionHomeSectionDelete = "content.home_section.delete"

	ActionFeatureCreate = "content.feature.create"
	ActionFeatureUpdate = "content.feature.update"
	ActionFeatureDelete = "content.feature.delete"

	ActionPricingPlanCreate = "content.pricing_plan.create"
	ActionPricingPlanUpdate = "content.pricing_plan.update"
	ActionPricingPlanDelete = "content.pricing_plan.delete"

	ActionPageCreate = "content.page.create"
	ActionPageUpdate = "content.page.update"
	ActionPageDelete = "content.page.delete"

	ActionDocCategoryCreate = "content.doc_category.create"
	ActionDocCategoryUpdate = "content.doc_category.update"
	ActionDocCategoryDelete = "content.doc_category.delete"

	ActionDocArticleCreate = "content.doc_article.create"
	ActionDocArticleUpdate = "content.doc_article.update"
	ActionDocArticleDelete = "content.doc_article.delete"

	ActionContactStatusUpdate = "contact.submission.update"

	ActionMediaUpload = "content.media.upload"
	ActionMediaDelete = "content.media.delete"
)

// Resource identifiers for content audit events.
const (
	ResourceSiteSettings      = "site_settings"
	ResourceNavigation        = "navigation_item"
	ResourceHomeSection       = "home_section"
	ResourceFeature           = "feature"
	ResourcePricingPlan       = "pricing_plan"
	ResourcePage              = "page"
	ResourceDocCategory       = "doc_category"
	ResourceDocArticle        = "doc_article"
	ResourceContactSubmission = "contact_submission"
	ResourceMedia             = "media_asset"
)

// safeDetailKeys allowlist: only these fields can be written into audit
// details. Bodies, secrets, and unknown nested objects are dropped.
var safeDetailKeys = map[string]struct{}{
	"slug":           {},
	"placement":      {},
	"visible":        {},
	"published":      {},
	"sort_order":     {},
	"highlighted":    {},
	"code":           {},
	"default_locale": {},
	"locales":        {},
	"category_id":    {},
	"parent_id":      {},
	"section_type":   {},
	"outcome_hint":   {},
	"status":         {},
	"mime":           {},
	"size_bytes":     {},
	"width":          {},
	"height":         {},
}

func redactDetails(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		if _, ok := safeDetailKeys[key]; !ok {
			continue
		}
		switch v := value.(type) {
		case []string:
			out[key] = append([]string(nil), v...)
		case string, bool, int, int64, float64:
			out[key] = v
		default:
			// Skip unexpected values to avoid leaking content bodies.
		}
	}
	return out
}

func mergeDetails(base, extra map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func (s *Service) createPendingEvent(ctx context.Context, actor Actor, action, resourceType, resourceID string, details map[string]any) (int64, error) {
	event := &adminauth.AuditEvent{
		ActorID:          actorIDPtr(actor.AdminID),
		ActorUsername:    actor.AdminUsername,
		ActorDisplayName: actor.AdminDisplayName,
		ActorRoleCodes:   append([]string(nil), actor.AdminRoleCodes...),
		Action:           action,
		ResourceType:     resourceType,
		ResourceID:       resourceID,
		Details:          redactDetails(details),
		RequestID:        actor.RequestID,
		SourceIP:         actor.SourceIP,
		UserAgent:        actor.UserAgent,
		EventAt:          time.Now().UTC(),
	}
	auditID, err := s.audits.CreatePending(ctx, event)
	if err != nil {
		return 0, apperr.New(503, apperr.CodeAuditUnavailable, "audit unavailable", fmt.Errorf("%w: %v", apperr.ErrAuditUnavailable, err))
	}
	return auditID, nil
}

func (s *Service) finalizeEvent(ctx context.Context, auditID int64, resourceID, outcome string, details map[string]any, action, resourceType string, actor Actor) {
	if err := s.audits.Finalize(ctx, auditID, resourceID, outcome, details); err != nil {
		s.logger.Error("content audit finalization failed; event left pending",
			"audit_id", auditID,
			"action", action,
			"resource_type", resourceType,
			"resource_id", resourceID,
			"request_id", actor.RequestID,
			"intended_outcome", outcome,
		)
	}
}

// withPendingAudit runs a content write under the pending-first audit protocol.
func (s *Service) withPendingAudit(ctx context.Context, actor Actor, action, resourceType, resourceID string, details map[string]any, fn func() error) error {
	auditID, err := s.createPendingEvent(ctx, actor, action, resourceType, resourceID, details)
	if err != nil {
		return err
	}
	writeErr := fn()
	outcome := adminauth.AuditOutcomeSucceeded
	finalDetails := redactDetails(details)
	if writeErr != nil {
		outcome = adminauth.AuditOutcomeFailed
		finalDetails = redactDetails(mergeDetails(details, map[string]any{"outcome_hint": "primary_write_failed"}))
	}
	s.finalizeEvent(ctx, auditID, resourceID, outcome, finalDetails, action, resourceType, actor)
	if writeErr != nil {
		return mapError(writeErr)
	}
	return nil
}

// createAudited runs the pending-first protocol for inserts whose id is only
// known after the write.
func (s *Service) createAudited(ctx context.Context, actor Actor, action, resourceType string, details map[string]any, fn func() (int64, error)) (int64, error) {
	auditID, err := s.createPendingEvent(ctx, actor, action, resourceType, "", details)
	if err != nil {
		return 0, err
	}
	id, writeErr := fn()
	outcome := adminauth.AuditOutcomeSucceeded
	resourceID := ""
	finalDetails := redactDetails(details)
	if writeErr != nil {
		outcome = adminauth.AuditOutcomeFailed
		finalDetails = redactDetails(mergeDetails(details, map[string]any{"outcome_hint": "primary_write_failed"}))
	} else if id > 0 {
		resourceID = idString(id)
	}
	s.finalizeEvent(ctx, auditID, resourceID, outcome, finalDetails, action, resourceType, actor)
	return id, mapError(writeErr)
}

func actorIDPtr(id int64) *int64 {
	if id <= 0 {
		return nil
	}
	copy := id
	return &copy
}
