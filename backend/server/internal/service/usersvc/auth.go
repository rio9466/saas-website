package usersvc

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	userdomain "github.com/rio9466/easy-admin/server/internal/domain/user"
	"github.com/rio9466/easy-admin/server/internal/platform/auth"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
)

// Authorization constants for business users.
const (
	AccessTokenTTL  = auth.AccessTokenTTL
	RefreshTokenTTL = auth.RefreshTokenTTL
)

// SessionResult carries tokens for the transport layer.
type SessionResult struct {
	AccessToken      string
	RefreshToken     string
	RefreshExpiresAt time.Time
	SessionID        string
}

// RegisterInput is the self-registration payload.
type RegisterInput struct {
	Username string
	Email    string
	Password string
	SourceIP string
}

// Register creates a business user from validated defaults, then activates it
// immediately (verification disabled) or leaves it pending and sends a
// one-time verification email (verification required). Registration is
// rate-limited and audited pending-first; a failed audit aborts the write.
func (s *Service) Register(ctx context.Context, meta Actor, in RegisterInput) (*userdomain.User, error) {
	username := strings.TrimSpace(in.Username)
	email := strings.TrimSpace(strings.ToLower(in.Email))
	ip := strings.TrimSpace(in.SourceIP)

	if err := s.allowRate(ctx, rateKeyRegister(ip), registerLimit, registerWindow); err != nil {
		return nil, err
	}

	settings, err := s.users.GetSystemSettings(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	if !settings.RegistrationEnabled {
		return nil, apperr.New(403, apperr.CodeRegistrationDisabled, "registration disabled", apperr.ErrRegistrationDisabled)
	}

	if err := validateUsername(username); err != nil {
		return nil, err
	}
	if err := validateEmail(email); err != nil {
		return nil, err
	}
	if len(in.Password) < 8 || len(in.Password) > 72 {
		return nil, apperr.New(400, apperr.CodeInvalidPassword, "invalid password", apperr.ErrInvalidPassword)
	}
	hash, err := s.passwords.Hash(in.Password)
	if err != nil {
		return nil, mapError(err)
	}

	now := time.Now().UTC()
	status := userdomain.StatusActive
	var emailVerifiedAt *time.Time
	if settings.EmailVerificationRequired {
		status = userdomain.StatusPendingVerification
	} else {
		emailVerifiedAt = &now
	}

	nu := &userdomain.User{
		Username:        username,
		Email:           email,
		PasswordHash:    hash,
		AvatarURL:       settings.DefaultAvatarURL,
		Nickname:        username,
		RegistrationIP:  ip,
		Status:          status,
		EmailVerifiedAt: emailVerifiedAt,
		LevelMode:       userdomain.LevelModeAuto,
	}

	userID, err := s.createAudited(ctx, meta, ActionUserRegister, ResourceUser, map[string]any{
		"username":            username,
		"email":               email,
		"registration_points": settings.RegistrationPoints.String(),
	}, func() (int64, error) {
		if err := s.users.CreateUser(ctx, nu, settings.DefaultLevelID); err != nil {
			return 0, err
		}
		return nu.ID, nil
	})
	if err != nil {
		return nil, err
	}

	// Grant the registration starting points exactly once via the immutable
	// ledger, keyed deterministically; a verification-disabled account gets
	// them immediately, a pending account on successful verification.
	if !settings.RegistrationPoints.IsZero() {
		if !settings.EmailVerificationRequired {
			if err := s.grantRegistrationPoints(ctx, userID, settings.RegistrationPoints); err != nil {
				s.rollbackRegistration(ctx, meta, userID, "registration_rolled_back")
				return nil, err
			}
		}
	}

	if settings.EmailVerificationRequired {
		if err := s.issueAndSendVerification(ctx, meta, settings, email, userID); err != nil {
			s.rollbackRegistration(ctx, meta, userID, "registration_rolled_back")
			return nil, err
		}
	}

	// Reload so the response carries the persisted level, timestamps, and
	// generated defaults instead of an unsaved in-memory shape.
	saved, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return nil, mapError(err)
	}
	return saved, nil
}

// grantRegistrationPoints applies the one-time registration bonus. Replays of
// the deterministic key are accepted as already-applied.
func (s *Service) grantRegistrationPoints(ctx context.Context, userID int64, points userdomain.Decimal4) error {
	key := "registration:" + idString(userID)
	_, err := s.users.AdjustPoints(ctx, userID, nil, points, userdomain.Zero(), "registration_starting_points", key)
	if errors.Is(err, primary.ErrIdempotencyReplay) {
		return nil
	}
	return mapError(err)
}

// Login authenticates with one identifier (username or email per enabled
// methods) plus password, records last_login_at, and issues a session.
func (s *Service) Login(ctx context.Context, meta Actor, identifier, password, sourceIP string) (*SessionResult, *userdomain.User, error) {
	identifier = strings.TrimSpace(identifier)
	if err := s.allowRate(ctx, rateKeyLoginByIP(sourceIP), loginLimit, loginWindow); err != nil {
		return nil, nil, err
	}
	if identifier == "" {
		if err := s.recordSecurityEvent(ctx, meta, ActionUserLogin, ResourceUser, "", map[string]any{"reason": "missing_credentials"}, false); err != nil {
			return nil, nil, err
		}
		return nil, nil, unauthorizedCredentials()
	}
	if err := s.allowRate(ctx, rateKeyLoginByID(identifier), loginLimit, loginWindow); err != nil {
		return nil, nil, err
	}

	settings, err := s.users.GetSystemSettings(ctx)
	if err != nil {
		return nil, nil, mapError(err)
	}

	method := loginMethodFor(identifier, settings.UsernameLoginEnabled, settings.EmailLoginEnabled)
	if method == "" {
		return nil, nil, apperr.New(400, apperr.CodeLoginMethodDisabled, "login method disabled", apperr.ErrLoginMethodDisabled)
	}

	var found *userdomain.User
	if method == "email" {
		found, err = s.users.GetUserByEmail(ctx, identifier)
	} else {
		found, err = s.users.GetUserByUsername(ctx, identifier)
	}
	if err != nil {
		if !errors.Is(err, primary.ErrNotFound) {
			return nil, nil, mapError(err)
		}
		// Anonymous credential burn so unknown identifiers cost the same as a
		// wrong password and never reveal whether the account exists.
		s.compareDummyPassword(password)
		if auditErr := s.recordSecurityEvent(ctx, meta, ActionUserLogin, ResourceUser, "", map[string]any{"reason": "invalid_credentials"}, false); auditErr != nil {
			return nil, nil, auditErr
		}
		return nil, nil, unauthorizedCredentials()
	}

	if err := s.passwords.Compare(found.PasswordHash, password); err != nil {
		actor := meta
		actor.ID = found.ID
		actor.Username = found.Username
		actor.Email = found.Email
		actor.Nickname = found.Nickname
		if auditErr := s.recordSecurityEvent(ctx, actor, ActionUserLogin, ResourceUser, idString(found.ID), map[string]any{
			"username": found.Username,
			"user_id":  idString(found.ID),
			"reason":   "invalid_credentials",
		}, false); auditErr != nil {
			return nil, nil, auditErr
		}
		return nil, nil, unauthorizedCredentials()
	}

	switch found.Status {
	case userdomain.StatusDisabled:
		actor := meta
		actor.ID = found.ID
		actor.Username = found.Username
		_ = s.recordSecurityEvent(ctx, actor, ActionUserLogin, ResourceUser, idString(found.ID), map[string]any{
			"reason": "account_disabled",
		}, false)
		return nil, nil, apperr.New(403, apperr.CodeAccountDisabled, "account disabled", apperr.ErrAccountDisabled)
	case userdomain.StatusPendingVerification:
		actor := meta
		actor.ID = found.ID
		actor.Username = found.Username
		_ = s.recordSecurityEvent(ctx, actor, ActionUserLogin, ResourceUser, idString(found.ID), map[string]any{
			"reason": "email_not_verified",
		}, false)
		return nil, nil, apperr.New(403, apperr.CodeEmailNotVerified, "email not verified", apperr.ErrEmailNotVerified)
	}

	// Successful login updates last_login_at; refresh and ordinary requests do not.
	if err := s.users.TouchLastLogin(ctx, found.ID); err != nil {
		return nil, nil, mapError(err)
	}

	actor := meta
	actor.ID = found.ID
	actor.Username = found.Username
	actor.Email = found.Email
	actor.Nickname = found.Nickname
	me, session, err := s.issueUserSession(ctx, found)
	if err != nil {
		return nil, nil, err
	}
	if err := s.recordSecurityEvent(ctx, actor, ActionUserLogin, ResourceUser, idString(found.ID), map[string]any{
		"username": found.Username,
		"user_id":  idString(found.ID),
	}, true); err != nil {
		_ = s.sessions.Revoke(ctx, session.SessionID)
		return nil, nil, err
	}
	return session, me, nil
}

// Refresh rotates the Redis user-session verifier and issues a new pair,
// honoring the session's absolute expiry (rotation never extends it).
func (s *Service) Refresh(ctx context.Context, meta Actor, refreshToken string) (*SessionResult, *userdomain.User, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, nil, apperr.New(401, apperr.CodeSessionInvalid, "session invalid", apperr.ErrSessionInvalid)
	}
	claims, err := s.tokens.ParseAndValidate(refreshToken, auth.TokenTypeRefresh)
	if err != nil {
		return nil, nil, mapError(err)
	}
	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || userID <= 0 {
		return nil, nil, apperr.New(401, apperr.CodeSessionInvalid, "session invalid", apperr.ErrSessionInvalid)
	}

	oldHash := auth.HashRefreshVerifier(claims.ID)
	u, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		_ = s.sessions.Revoke(ctx, claims.SessionID)
		return nil, nil, mapError(err)
	}
	switch u.Status {
	case userdomain.StatusDisabled:
		_ = s.sessions.Revoke(ctx, claims.SessionID)
		return nil, nil, apperr.New(403, apperr.CodeAccountDisabled, "account disabled", apperr.ErrAccountDisabled)
	case userdomain.StatusPendingVerification:
		_ = s.sessions.Revoke(ctx, claims.SessionID)
		return nil, nil, apperr.New(403, apperr.CodeEmailNotVerified, "email not verified", apperr.ErrEmailNotVerified)
	}

	sess, err := s.sessions.Get(ctx, claims.SessionID)
	if err != nil {
		return nil, nil, mapError(err)
	}
	if sess.AdminID != userID {
		_ = s.sessions.Revoke(ctx, claims.SessionID)
		return nil, nil, apperr.New(401, apperr.CodeSessionInvalid, "session invalid", apperr.ErrSessionInvalid)
	}
	if sess.AuthEpoch != u.AuthEpoch {
		_ = s.sessions.Revoke(ctx, claims.SessionID)
		return nil, nil, apperr.New(401, apperr.CodeSessionInvalid, "session invalid", apperr.ErrSessionInvalid)
	}

	absoluteExpiry := sess.ExpiresAt.UTC()
	newRefresh, newRefreshClaims, err := s.tokens.IssueRefreshExpiring(strconv.FormatInt(userID, 10), claims.SessionID, absoluteExpiry)
	if err != nil {
		_ = s.sessions.Revoke(ctx, claims.SessionID)
		return nil, nil, mapError(err)
	}
	if err := s.sessions.RotateRefresh(ctx, claims.SessionID, oldHash, auth.HashRefreshVerifier(newRefreshClaims.ID)); err != nil {
		if errors.Is(err, auth.ErrRefreshReplay) {
			actor := meta
			actor.ID = u.ID
			actor.Username = u.Username
			actor.Email = u.Email
			actor.Nickname = u.Nickname
			actor.SessionID = claims.SessionID
			if auditErr := s.recordSecurityEvent(ctx, actor, ActionUserRefreshReplay, ResourceUserSession, claims.SessionID, map[string]any{
				"user_id": idString(u.ID),
				"reason":  "refresh_replay",
			}, false); auditErr != nil {
				return nil, nil, auditErr
			}
			return nil, nil, mapError(err)
		}
		return nil, nil, mapError(err)
	}

	accessToken, _, err := s.tokens.IssueAccess(strconv.FormatInt(userID, 10), claims.SessionID)
	if err != nil {
		_ = s.sessions.Revoke(ctx, claims.SessionID)
		return nil, nil, mapError(err)
	}
	// The client reloads /me after a refresh; no profile is attached here so
	// refresh never requires (or updates) session-scoped actor state.
	return &SessionResult{
		AccessToken:      accessToken,
		RefreshToken:     newRefresh,
		RefreshExpiresAt: absoluteExpiry,
		SessionID:        claims.SessionID,
	}, nil, nil
}

// Logout revokes the current user session under the pending-first audit policy.
func (s *Service) Logout(ctx context.Context, actor Actor) error {
	if err := s.requireAuth(actor); err != nil {
		return err
	}
	return s.withPendingAudit(ctx, actor, ActionUserLogout, ResourceUserSession, actor.SessionID, map[string]any{
		"user_id":  idString(actor.ID),
		"username": actor.Username,
	}, func() error {
		return s.sessions.Revoke(ctx, actor.SessionID)
	})
}

// Me returns the authenticated user profile with live data.
func (s *Service) Me(ctx context.Context, actor Actor, userID int64) (*userdomain.User, error) {
	if err := s.requireAuth(actor); err != nil {
		return nil, err
	}
	if userID <= 0 {
		userID = actor.ID
	}
	u, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return nil, mapError(err)
	}
	if u.Status != userdomain.StatusActive {
		return nil, apperr.New(403, apperr.CodeAccountDisabled, "account disabled", apperr.ErrAccountDisabled)
	}
	return u, nil
}

func toMeResult(u *userdomain.User) *userdomain.User {
	return u
}

func (s *Service) issueUserSession(ctx context.Context, u *userdomain.User) (*userdomain.User, *SessionResult, error) {
	sessionID := uuid.NewString()
	subject := strconv.FormatInt(u.ID, 10)

	refreshToken, refreshClaims, err := s.tokens.IssueRefresh(subject, sessionID)
	if err != nil {
		return nil, nil, mapError(err)
	}
	accessToken, _, err := s.tokens.IssueAccess(subject, sessionID)
	if err != nil {
		return nil, nil, mapError(err)
	}
	expiresAt := refreshClaims.ExpiresAt.Time.UTC()
	if err := s.sessions.Create(ctx, sessionID, u.ID, auth.HashRefreshVerifier(refreshClaims.ID), u.AuthEpoch, expiresAt); err != nil {
		return nil, nil, mapError(err)
	}
	return u, &SessionResult{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: expiresAt,
		SessionID:        sessionID,
	}, nil
}

// loginMethodFor decides email vs username login from the enabled switches.
// When both are enabled an @-containing identifier is email; username syntax
// never contains '@' so the modes cannot overlap. An empty result means the
// identifier does not match any enabled method.
func loginMethodFor(identifier string, usernameEnabled, emailEnabled bool) string {
	hasAt := strings.Contains(identifier, "@")
	switch {
	case hasAt && emailEnabled:
		return "email"
	case !hasAt && usernameEnabled:
		return "username"
	default:
		return ""
	}
}

// deleteUserBestEffort removes a user that failed to finish registration
// (verification email could not be sent after creation). Only freshly created
// rows are safe to delete; the error is logged without identity details.
func (s *Service) deleteUserBestEffort(ctx context.Context, userID int64) error {
	return s.users.DeleteUserBestEffort(ctx, userID)
}

// rollbackRegistration undoes a freshly created account whose registration
// could not complete (starting-points grant or verification email failed).
// The create step is already audited as succeeded, so a matching failed
// security event is recorded; otherwise the audit trail would claim a
// registration that no longer exists. Details stay generic and never reveal
// whether an address exists.
func (s *Service) rollbackRegistration(ctx context.Context, meta Actor, userID int64, reason string) {
	if err := s.deleteUserBestEffort(ctx, userID); err != nil {
		s.logger.Error("user registration rollback failed",
			"user_id", idString(userID),
			"request_id", meta.RequestID,
		)
	}
	_ = s.recordSecurityEvent(ctx, meta, ActionUserRegister, ResourceUser, idString(userID), map[string]any{
		"user_id": idString(userID),
		"reason":  reason,
	}, false)
}
