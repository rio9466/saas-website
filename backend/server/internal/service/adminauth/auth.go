package adminauth

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	"github.com/rio9466/easy-admin/server/internal/platform/auth"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
)

// LoginResult carries tokens for the transport layer.
// AccessToken belongs in JSON; RefreshToken belongs only in an HttpOnly cookie.
type LoginResult struct {
	AccessToken      string
	RefreshToken     string
	RefreshExpiresAt time.Time
	SessionID        string
	Admin            MeResult
}

// MeResult is the current administrator profile with live authorization codes.
type MeResult struct {
	ID              int64
	Username        string
	DisplayName     string
	Enabled         bool
	RoleCodes       []string
	PermissionCodes []string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Login authenticates credentials, creates a Redis session, and issues tokens.
// Security events are fail-closed: audit must succeed before a session is returned,
// and failed authentications still require a durable audit record.
func (s *Service) Login(ctx context.Context, meta Actor, username, password string) (*LoginResult, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		if err := s.recordSecurityEvent(ctx, meta, ActionAuthLogin, ResourceAuth, "", map[string]any{
			"username": username,
			"reason":   "missing_credentials",
		}, false); err != nil {
			return nil, err
		}
		return nil, unauthorizedCredentials()
	}

	admin, err := s.admins.GetAdministratorByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, primary.ErrNotFound) {
			// Burn one configured-cost bcrypt comparison so a nonexistent
			// username costs the same as a wrong password and does not reveal
			// whether the account exists.
			s.compareDummyPassword(password)
			if auditErr := s.recordSecurityEvent(ctx, meta, ActionAuthLogin, ResourceAuth, "", map[string]any{
				"username": username,
				"reason":   "invalid_credentials",
			}, false); auditErr != nil {
				return nil, auditErr
			}
			return nil, unauthorizedCredentials()
		}
		return nil, mapError(err)
	}

	if err := s.passwords.Compare(admin.PasswordHash, password); err != nil {
		actor := meta
		actor.ID = admin.ID
		actor.Username = admin.Username
		actor.DisplayName = admin.DisplayName
		actor.RoleCodes = cloneStrings(admin.RoleCodes)
		if auditErr := s.recordSecurityEvent(ctx, actor, ActionAuthLogin, ResourceAdministrator, idString(admin.ID), map[string]any{
			"username":         admin.Username,
			"administrator_id": idString(admin.ID),
			"reason":           "invalid_credentials",
		}, false); auditErr != nil {
			return nil, auditErr
		}
		return nil, unauthorizedCredentials()
	}

	if !admin.Enabled {
		actor := meta
		actor.ID = admin.ID
		actor.Username = admin.Username
		actor.DisplayName = admin.DisplayName
		actor.RoleCodes = cloneStrings(admin.RoleCodes)
		if auditErr := s.recordSecurityEvent(ctx, actor, ActionAuthLogin, ResourceAdministrator, idString(admin.ID), map[string]any{
			"username":         admin.Username,
			"administrator_id": idString(admin.ID),
			"reason":           "account_disabled",
		}, false); auditErr != nil {
			return nil, auditErr
		}
		return nil, apperr.New(403, apperr.CodeAccountDisabled, "account disabled", apperr.ErrAccountDisabled)
	}

	result, err := s.issueSession(ctx, admin)
	if err != nil {
		return nil, err
	}

	actor := meta
	actor.ID = admin.ID
	actor.Username = admin.Username
	actor.DisplayName = admin.DisplayName
	actor.RoleCodes = cloneStrings(admin.RoleCodes)
	actor.SessionID = result.SessionID
	if err := s.recordSecurityEvent(ctx, actor, ActionAuthLogin, ResourceAdministrator, idString(admin.ID), map[string]any{
		"username":         admin.Username,
		"administrator_id": idString(admin.ID),
		"role_codes":       cloneStrings(admin.RoleCodes),
	}, true); err != nil {
		_ = s.sessions.Revoke(ctx, result.SessionID)
		return nil, err
	}

	return result, nil
}

// Refresh validates a refresh token, rotates the Redis verifier, and issues a new pair.
func (s *Service) Refresh(ctx context.Context, meta Actor, refreshToken string) (*LoginResult, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, apperr.New(401, apperr.CodeSessionInvalid, "session invalid", apperr.ErrSessionInvalid)
	}

	claims, err := s.tokens.ParseAndValidate(refreshToken, auth.TokenTypeRefresh)
	if err != nil {
		return nil, mapError(err)
	}

	adminID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || adminID <= 0 {
		return nil, apperr.New(401, apperr.CodeSessionInvalid, "session invalid", apperr.ErrSessionInvalid)
	}

	oldHash := auth.HashRefreshVerifier(claims.ID)
	admin, err := s.admins.GetAdministratorByID(ctx, adminID)
	if err != nil {
		_ = s.sessions.Revoke(ctx, claims.SessionID)
		return nil, mapError(err)
	}
	if !admin.Enabled {
		_ = s.sessions.Revoke(ctx, claims.SessionID)
		return nil, apperr.New(403, apperr.CodeAccountDisabled, "account disabled", apperr.ErrAccountDisabled)
	}

	sess, err := s.sessions.Get(ctx, claims.SessionID)
	if err != nil {
		return nil, mapError(err)
	}
	if sess.AdminID != adminID {
		_ = s.sessions.Revoke(ctx, claims.SessionID)
		return nil, apperr.New(401, apperr.CodeSessionInvalid, "session invalid", apperr.ErrSessionInvalid)
	}
	if sess.AuthEpoch != admin.AuthEpoch {
		// Password change/reset or account enable/disable invalidated every
		// session issued under the old auth epoch.
		_ = s.sessions.Revoke(ctx, claims.SessionID)
		return nil, apperr.New(401, apperr.CodeSessionInvalid, "session invalid", apperr.ErrSessionInvalid)
	}

	subject := strconv.FormatInt(admin.ID, 10)
	// Rotation honors the session's absolute expiry: the new refresh JWT exp
	// equals the original login-time expiry and Redis TTLs are recomputed from
	// it, so repeated refreshes never extend the session lifetime.
	absoluteExpiry := sess.ExpiresAt.UTC()
	newRefresh, newRefreshClaims, err := s.tokens.IssueRefreshExpiring(subject, claims.SessionID, absoluteExpiry)
	if err != nil {
		_ = s.sessions.Revoke(ctx, claims.SessionID)
		return nil, mapError(err)
	}
	newHash := auth.HashRefreshVerifier(newRefreshClaims.ID)
	expiresAt := absoluteExpiry

	if err := s.sessions.RotateRefresh(ctx, claims.SessionID, oldHash, newHash); err != nil {
		if errors.Is(err, auth.ErrRefreshReplay) {
			actor := meta
			actor.ID = admin.ID
			actor.Username = admin.Username
			actor.DisplayName = admin.DisplayName
			actor.RoleCodes = cloneStrings(admin.RoleCodes)
			actor.SessionID = claims.SessionID
			if auditErr := s.recordSecurityEvent(ctx, actor, ActionAuthRefreshReplay, ResourceSession, claims.SessionID, map[string]any{
				"administrator_id": idString(admin.ID),
				"username":         admin.Username,
				"reason":           "refresh_replay",
			}, false); auditErr != nil {
				return nil, auditErr
			}
			return nil, mapError(err)
		}
		return nil, mapError(err)
	}

	accessToken, _, err := s.tokens.IssueAccess(subject, claims.SessionID)
	if err != nil {
		_ = s.sessions.Revoke(ctx, claims.SessionID)
		return nil, mapError(err)
	}

	perms, err := s.admins.EffectivePermissionCodes(ctx, admin.ID)
	if err != nil {
		return nil, mapError(err)
	}

	return &LoginResult{
		AccessToken:      accessToken,
		RefreshToken:     newRefresh,
		RefreshExpiresAt: expiresAt,
		SessionID:        claims.SessionID,
		Admin: MeResult{
			ID:              admin.ID,
			Username:        admin.Username,
			DisplayName:     admin.DisplayName,
			Enabled:         admin.Enabled,
			RoleCodes:       cloneStrings(admin.RoleCodes),
			PermissionCodes: perms,
			CreatedAt:       admin.CreatedAt,
			UpdatedAt:       admin.UpdatedAt,
		},
	}, nil
}

// Logout revokes the current Redis session after a durable audit pending record exists.
func (s *Service) Logout(ctx context.Context, actor Actor) error {
	if err := s.requireAuth(actor); err != nil {
		return err
	}
	return s.withPendingAudit(ctx, actor, ActionAuthLogout, ResourceSession, actor.SessionID, map[string]any{
		"administrator_id": idString(actor.ID),
		"username":         actor.Username,
	}, func() error {
		return s.sessions.Revoke(ctx, actor.SessionID)
	})
}

// Me returns the current administrator with live roles and permissions.
func (s *Service) Me(ctx context.Context, actor Actor) (*MeResult, error) {
	if err := s.requireAuth(actor); err != nil {
		return nil, err
	}
	admin, err := s.admins.GetAdministratorByID(ctx, actor.ID)
	if err != nil {
		return nil, mapError(err)
	}
	if !admin.Enabled {
		return nil, apperr.New(403, apperr.CodeAccountDisabled, "account disabled", apperr.ErrAccountDisabled)
	}
	perms, err := s.admins.EffectivePermissionCodes(ctx, admin.ID)
	if err != nil {
		return nil, mapError(err)
	}
	return &MeResult{
		ID:              admin.ID,
		Username:        admin.Username,
		DisplayName:     admin.DisplayName,
		Enabled:         admin.Enabled,
		RoleCodes:       cloneStrings(admin.RoleCodes),
		PermissionCodes: perms,
		CreatedAt:       admin.CreatedAt,
		UpdatedAt:       admin.UpdatedAt,
	}, nil
}

// UpdateMyProfile updates the current administrator's display name only. Any
// authenticated, enabled administrator may change their own display name; no
// admin.user.update permission is required and only the current account's
// display_name may be changed. The write uses the pending-first audit protocol
// with action administrator.profile_update and fails closed when the log store
// is unavailable. Roles, permissions, enabled state, and the auth epoch are
// unchanged, so the current session stays valid.
func (s *Service) UpdateMyProfile(ctx context.Context, actor Actor, displayName string) (*MeResult, error) {
	if err := s.requireAuth(actor); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(displayName)
	if name == "" {
		return nil, validationError("display name is required")
	}

	admin, err := s.admins.GetAdministratorByID(ctx, actor.ID)
	if err != nil {
		return nil, mapError(err)
	}
	if !admin.Enabled {
		return nil, apperr.New(403, apperr.CodeAccountDisabled, "account disabled", apperr.ErrAccountDisabled)
	}

	// No-op (same name): skip the write and the audit row, return current profile.
	if admin.DisplayName != name {
		err = s.withPendingAudit(ctx, actor, ActionAdminProfileUpdate, ResourceAdministrator, idString(actor.ID), map[string]any{
			"administrator_id": idString(actor.ID),
			"username":         admin.Username,
			"display_name":     name,
		}, func() error {
			return s.admins.UpdateAdministrator(ctx, actor.ID, &name)
		})
		if err != nil {
			return nil, err
		}
	}
	return s.Me(ctx, actor)
}

// ChangePassword verifies the current password, stores a new hash, and revokes all sessions.
func (s *Service) ChangePassword(ctx context.Context, actor Actor, currentPassword, newPassword string) error {
	if err := s.requireAuth(actor); err != nil {
		return err
	}
	if strings.TrimSpace(currentPassword) == "" || strings.TrimSpace(newPassword) == "" {
		return validationError("current and new password are required")
	}

	admin, err := s.admins.GetAdministratorByUsername(ctx, actor.Username)
	if err != nil {
		byID, idErr := s.admins.GetAdministratorByID(ctx, actor.ID)
		if idErr != nil {
			return mapError(err)
		}
		admin, err = s.admins.GetAdministratorByUsername(ctx, byID.Username)
		if err != nil {
			return mapError(err)
		}
	}

	if err := s.passwords.Compare(admin.PasswordHash, currentPassword); err != nil {
		return apperr.New(400, apperr.CodeInvalidPassword, "invalid password", apperr.ErrInvalidPassword)
	}
	if err := auth.ValidatePasswordLength(newPassword); err != nil {
		return mapError(err)
	}
	if currentPassword == newPassword {
		return validationError("new password must differ from current password")
	}

	hash, err := s.passwords.Hash(newPassword)
	if err != nil {
		return mapError(err)
	}

	// The repository write bumps auth_epoch, invalidating every session
	// (including the current one) immediately. Redis revocation is a
	// best-effort cleanup after the audited primary write succeeds.
	if err := s.withPendingAudit(ctx, actor, ActionAuthChangePassword, ResourceAdministrator, idString(actor.ID), map[string]any{
		"administrator_id": idString(actor.ID),
		"username":         admin.Username,
	}, func() error {
		return s.admins.SetAdministratorPassword(ctx, actor.ID, hash)
	}); err != nil {
		return err
	}
	s.revokeSessionsBestEffort(ctx, actor.ID, "password change")
	return nil
}

func (s *Service) issueSession(ctx context.Context, admin *adminauth.Administrator) (*LoginResult, error) {
	sessionID := uuid.NewString()
	subject := strconv.FormatInt(admin.ID, 10)

	refreshToken, refreshClaims, err := s.tokens.IssueRefresh(subject, sessionID)
	if err != nil {
		return nil, mapError(err)
	}
	accessToken, _, err := s.tokens.IssueAccess(subject, sessionID)
	if err != nil {
		return nil, mapError(err)
	}

	expiresAt := refreshClaims.ExpiresAt.Time.UTC()
	if err := s.sessions.Create(ctx, sessionID, admin.ID, auth.HashRefreshVerifier(refreshClaims.ID), admin.AuthEpoch, expiresAt); err != nil {
		return nil, mapError(err)
	}

	perms, err := s.admins.EffectivePermissionCodes(ctx, admin.ID)
	if err != nil {
		_ = s.sessions.Revoke(ctx, sessionID)
		return nil, mapError(err)
	}

	return &LoginResult{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: expiresAt,
		SessionID:        sessionID,
		Admin: MeResult{
			ID:              admin.ID,
			Username:        admin.Username,
			DisplayName:     admin.DisplayName,
			Enabled:         admin.Enabled,
			RoleCodes:       cloneStrings(admin.RoleCodes),
			PermissionCodes: perms,
			CreatedAt:       admin.CreatedAt,
			UpdatedAt:       admin.UpdatedAt,
		},
	}, nil
}
