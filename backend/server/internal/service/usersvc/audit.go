package usersvc

import (
	"context"
	"fmt"
	"time"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
)

// Audit action identifiers for business-user events.
const (
	ActionUserRegister           = "user.register"
	ActionUserLogin              = "user.login"
	ActionUserLogout             = "user.logout"
	ActionUserRefreshReplay      = "user.refresh_replay"
	ActionUserVerifyEmail        = "user.verify_email"
	ActionUserResendVerification = "user.resend_verification"
	ActionUserCreate             = "user.create"
	ActionUserUpdate             = "user.update"
	ActionUserEnable             = "user.enable"
	ActionUserDisable            = "user.disable"
	ActionUserResetPassword      = "user.reset_password"
	ActionUserPointsAdjust       = "user.points_adjust"
	ActionUserLevelAssign        = "user.level_assign"
	ActionUserLevelCreate        = "user_level.create"
	ActionUserLevelUpdate        = "user_level.update"
	ActionSystemSettingsUpdate   = "system_settings.update"
)

// Resource identifiers for business-user audit events.
const (
	ResourceUser              = "user"
	ResourceUserLevel         = "user_level"
	ResourceSystemSettings    = "system_settings"
	ResourceUserSession       = "user_session"
	ResourceEmailVerification = "email_verification"
)

// safeDetailKeys allowlist: only these fields can ever be written into audit
// details. Passwords, tokens, SMTP secrets, cookies, and full bodies are
// rejected by the allowlist itself.
var safeDetailKeys = map[string]struct{}{
	"username":                    {},
	"email":                       {},
	"nickname":                    {},
	"avatar_url":                  {},
	"reason":                      {},
	"outcome_hint":                {},
	"user_id":                     {},
	"target_username":             {},
	"new_status":                  {},
	"old_status":                  {},
	"level_id":                    {},
	"level_code":                  {},
	"level_name":                  {},
	"threshold_points":            {},
	"sort_order":                  {},
	"enabled":                     {},
	"points_delta":                {},
	"consumption_delta":           {},
	"balance_after":               {},
	"consumption_after":           {},
	"idempotency_key":             {},
	"default_level_id":            {},
	"platform_name":               {},
	"registration_enabled":        {},
	"username_login_enabled":      {},
	"email_login_enabled":         {},
	"email_verification_required": {},
	"registration_points":         {},
	"smtp_enabled":                {},
	"smtp_host":                   {},
	"smtp_port":                   {},
	"smtp_username":               {},
	"smtp_from_email":             {},
	"smtp_from_name":              {},
	"smtp_tls_mode":               {},
	"password_configured":         {},
	"administrator_id":            {},
	"role_codes":                  {},
	"display_name":                {},
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
		default:
			// Skip unexpected nested objects to avoid leaking secrets.
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

func actorAuditPtr(id int64) *int64 {
	if id <= 0 {
		return nil
	}
	copy := id
	return &copy
}

// createPendingEvent inserts a pending audit record; failure blocks the
// security-sensitive primary write (mandatory-audit policy). Admin-facing
// operations record the administrator actor; user-facing auth events record
// the business-user actor with the "user" role marker.
func (s *Service) createPendingEvent(ctx context.Context, actor Actor, action, resourceType, resourceID string, details map[string]any) (int64, error) {
	auditActorID, username, displayName, roleCodes := actor.ID, actor.Username, actor.Nickname, []string{"user"}
	if actor.AdminID > 0 {
		auditActorID = actor.AdminID
		username = actor.AdminUsername
		displayName = actor.AdminDisplayName
		roleCodes = cloneStrings(actor.AdminRoleCodes)
	}
	event := &adminauth.AuditEvent{
		ActorID:          actorAuditPtr(auditActorID),
		ActorUsername:    username,
		ActorDisplayName: displayName,
		ActorRoleCodes:   roleCodes,
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

// finalizeEvent marks a pending event succeeded/failed. Failures leave a
// diagnosable pending row and are logged sanitized; the two databases cannot
// be updated atomically.
func (s *Service) finalizeEvent(ctx context.Context, auditID int64, resourceID, outcome string, details map[string]any, action, resourceType, requestID string) {
	if err := s.audits.Finalize(ctx, auditID, resourceID, outcome, details); err != nil {
		s.logger.Error("user audit finalization failed; event left pending",
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
	s.finalizeEvent(ctx, auditID, resourceID, outcome, finalDetails, action, resourceType, actor.RequestID)
	if writeErr != nil {
		return mapError(writeErr)
	}
	return nil
}

// createAudited runs the pending-first protocol for writes whose resource id is
// only known after the primary insert.
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
	s.finalizeEvent(ctx, auditID, resourceID, outcome, finalDetails, action, resourceType, actor.RequestID)
	return id, mapError(writeErr)
}

// recordSecurityEvent writes a pending-then-final audit for auth/security
// events and fails closed: callers must abort when the pending insert fails.
func (s *Service) recordSecurityEvent(ctx context.Context, actor Actor, action, resourceType, resourceID string, details map[string]any, succeeded bool) error {
	auditID, err := s.createPendingEvent(ctx, actor, action, resourceType, resourceID, details)
	if err != nil {
		s.logger.Error("user security audit create failed",
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
