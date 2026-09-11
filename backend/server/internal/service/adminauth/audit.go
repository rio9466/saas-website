package adminauth

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	"github.com/rio9466/easy-admin/server/internal/repository/logdb"
)

// Audit action identifiers written to the log database.
const (
	ActionAuthLogin          = "auth.login"
	ActionAuthLogout         = "auth.logout"
	ActionAuthRefreshReplay  = "auth.refresh_replay"
	ActionAuthChangePassword = "auth.change_password"
	ActionAdminCreate        = "administrator.create"
	ActionAdminUpdate        = "administrator.update"
	ActionAdminEnable        = "administrator.enable"
	ActionAdminDisable       = "administrator.disable"
	ActionAdminResetPassword = "administrator.reset_password"
	ActionAdminAssignRoles   = "administrator.assign_roles"
	ActionAdminProfileUpdate = "administrator.profile_update"
	ActionRoleCreate         = "role.create"
	ActionRoleUpdate         = "role.update"
	ActionRoleAssignPerms    = "role.assign_permissions"
)

// Resource type identifiers for audit events.
const (
	ResourceAdministrator = "administrator"
	ResourceRole          = "role"
	ResourceSession       = "session"
	ResourceAuth          = "auth"
)

// Safe detail keys allowed in audit payloads.
var safeDetailKeys = map[string]struct{}{
	"username":         {},
	"display_name":     {},
	"name":             {},
	"description":      {},
	"enabled":          {},
	"role_codes":       {},
	"permission_codes": {},
	"administrator_id": {},
	"role_id":          {},
	"role_code":        {},
	"target_username":  {},
	"reason":           {},
	"outcome_hint":     {},
}

// ListAuditEvents returns a filtered audit page. Requires audit.log.read.
func (s *Service) ListAuditEvents(ctx context.Context, actor Actor, filter logdb.AuditListFilter) (adminauth.Page[adminauth.AuditEvent], error) {
	if err := s.requireAuth(actor); err != nil {
		return adminauth.Page[adminauth.AuditEvent]{}, err
	}
	if err := s.requirePermission(ctx, actor, adminauth.PermAuditLogRead); err != nil {
		return adminauth.Page[adminauth.AuditEvent]{}, err
	}
	page, err := s.audits.List(ctx, filter)
	if err != nil {
		return adminauth.Page[adminauth.AuditEvent]{}, mapError(err)
	}
	return page, nil
}

// GetAuditEvent returns one audit event by ID. Requires audit.log.read.
func (s *Service) GetAuditEvent(ctx context.Context, actor Actor, id int64) (*adminauth.AuditEvent, error) {
	if err := s.requireAuth(actor); err != nil {
		return nil, err
	}
	if err := s.requirePermission(ctx, actor, adminauth.PermAuditLogRead); err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, validationError("invalid audit event id")
	}
	event, err := s.audits.GetByID(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	return event, nil
}

// redactDetails keeps only allowlisted safe fields for audit storage.
func redactDetails(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		if _, ok := safeDetailKeys[key]; !ok {
			continue
		}
		switch v := value.(type) {
		case []string:
			out[key] = cloneStrings(v)
		case string, bool, int, int64, float64:
			out[key] = v
		case fmt.Stringer:
			out[key] = v.String()
		default:
			// Skip unexpected nested objects to avoid leaking secrets.
		}
	}
	return out
}

func idString(id int64) string {
	return strconv.FormatInt(id, 10)
}

func actorPtr(actor Actor) *int64 {
	if actor.ID <= 0 {
		return nil
	}
	id := actor.ID
	return &id
}

// createPendingEvent inserts a pending audit record and returns its id. A
// failure is a mandatory-audit failure: callers must not proceed with the
// security-sensitive primary write.
func (s *Service) createPendingEvent(ctx context.Context, actor Actor, action, resourceType, resourceID string, details map[string]any) (int64, error) {
	event := &adminauth.AuditEvent{
		ActorID:          actorPtr(actor),
		ActorUsername:    actor.Username,
		ActorDisplayName: actor.DisplayName,
		ActorRoleCodes:   cloneStrings(actor.RoleCodes),
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

// finalizeEvent marks a pending event succeeded/failed and may attach a
// resource ID that only became known after the primary write (creates). A
// finalization failure leaves a diagnosable pending row and is logged without
// secrets; the two databases cannot be updated in one distributed transaction.
func (s *Service) finalizeEvent(ctx context.Context, auditID int64, resourceID, outcome string, details map[string]any, action, resourceType, requestID string) {
	if err := s.audits.Finalize(ctx, auditID, resourceID, outcome, details); err != nil {
		s.logger.Error("audit finalization failed; event left pending",
			"audit_id", auditID,
			"action", action,
			"resource_type", resourceType,
			"resource_id", resourceID,
			"request_id", requestID,
			"intended_outcome", outcome,
		)
	}
}

// withPendingAudit runs a primary write under the pending-first audit protocol.
// If CreatePending fails, the primary write is not executed.
func (s *Service) withPendingAudit(
	ctx context.Context,
	actor Actor,
	action, resourceType, resourceID string,
	details map[string]any,
	fn func() error,
) error {
	auditID, err := s.createPendingEvent(ctx, actor, action, resourceType, resourceID, details)
	if err != nil {
		return err
	}

	writeErr := fn()
	outcome := adminauth.AuditOutcomeSucceeded
	finalDetails := redactDetails(details)
	if writeErr != nil {
		outcome = adminauth.AuditOutcomeFailed
		finalDetails = redactDetails(mergeDetails(details, map[string]any{
			"outcome_hint": "primary_write_failed",
		}))
	}

	s.finalizeEvent(ctx, auditID, resourceID, outcome, finalDetails, action, resourceType, actor.RequestID)
	if writeErr != nil {
		return mapError(writeErr)
	}
	return nil
}

// createAudited runs the pending-first audit protocol for a create whose
// decimal-string resource ID is only known after the primary insert. The
// pending event starts with an empty resource ID and is finalized with the
// created ID on success; a failed insert is finalized as failed.
func (s *Service) createAudited(
	ctx context.Context,
	actor Actor,
	action, resourceType string,
	details map[string]any,
	fn func() (int64, error),
) (int64, error) {
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
		finalDetails = redactDetails(mergeDetails(details, map[string]any{
			"outcome_hint": "primary_write_failed",
		}))
	} else if id > 0 {
		resourceID = idString(id)
	}

	s.finalizeEvent(ctx, auditID, resourceID, outcome, finalDetails, action, resourceType, actor.RequestID)
	return id, mapError(writeErr)
}

// recordSecurityEvent writes a pending-then-final audit for auth/security events.
// Audit failure is logged; callers decide whether to fail closed.
func (s *Service) recordSecurityEvent(
	ctx context.Context,
	actor Actor,
	action, resourceType, resourceID string,
	details map[string]any,
	succeeded bool,
) error {
	auditID, err := s.createPendingEvent(ctx, actor, action, resourceType, resourceID, details)
	if err != nil {
		s.logger.Error("security audit create failed",
			"action", action,
			"resource_type", resourceType,
			"resource_id", resourceID,
			"request_id", actor.RequestID,
		)
		return err
	}

	outcome := adminauth.AuditOutcomeSucceeded
	if !succeeded {
		outcome = adminauth.AuditOutcomeFailed
	}
	s.finalizeEvent(ctx, auditID, resourceID, outcome, redactDetails(details), action, resourceType, actor.RequestID)
	return nil
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
