package usersvc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	userdomain "github.com/rio9466/easy-admin/server/internal/domain/user"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
)

// Password-reset token Redis namespace. Tokens are stored hashed (sha256) with
// a short TTL, exactly like email verification tokens.
const userResetKeyPrefix = "easy-admin:user-reset:"

func userResetKey(userID int64) string {
	return userResetKeyPrefix + idString(userID)
}

// ForgotPassword issues a one-time password-reset token for an existing
// address. The response is identical whether or not the address exists (the
// caller always returns 200 for valid input), and an SMTP failure is logged
// and swallowed for the same reason. Only rate limiting (fail closed) and
// input validation surface as errors.
func (s *Service) ForgotPassword(ctx context.Context, actor Actor, email string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if err := validateEmail(email); err != nil {
		return err
	}
	// Per-IP and per-address fixed windows: three requests per hour each.
	if err := s.allowRate(ctx, rateKeyForgotByIP(actor.SourceIP), forgotLimit, forgotWindow); err != nil {
		return err
	}
	if err := s.allowRate(ctx, rateKeyForgotByEmail(email), forgotLimit, forgotWindow); err != nil {
		return err
	}

	settings, err := s.users.GetSystemSettings(ctx)
	if err != nil {
		return mapError(err)
	}
	u, err := s.users.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, primary.ErrNotFound) {
			return nil
		}
		return mapError(err)
	}

	token, err := generateVerificationToken()
	if err != nil {
		return mapError(err)
	}
	if err := s.storeResetToken(ctx, u.ID, hashVerificationToken(token)); err != nil {
		return mapError(err)
	}
	if err := s.sendResetEmail(ctx, settings, u.Email, token); err != nil {
		// Never reveal delivery outcome or account existence to the caller.
		s.logger.Error("password reset email send failed",
			"user_id", idString(u.ID),
			"request_id", actor.RequestID,
		)
	}
	return nil
}

// ResetPassword redeems a one-time reset token and stores a new password. The
// token is atomically consumed (single-use); success bumps the user's
// auth_epoch and revokes every session.
func (s *Service) ResetPassword(ctx context.Context, actor Actor, email, token, newPassword string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	token = strings.TrimSpace(token)
	if email == "" || token == "" {
		return verificationInvalid()
	}
	if len(newPassword) < 8 || len(newPassword) > 72 {
		return validationError("password must be 8-72 bytes")
	}
	if err := s.allowRate(ctx, rateKeyResetByIP(actor.SourceIP), resetLimit, resetWindow); err != nil {
		return err
	}

	u, err := s.users.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, primary.ErrNotFound) {
			return verificationInvalid()
		}
		return mapError(err)
	}

	// Atomic one-time consumption before any account mutation. A replayed or
	// wrong token can never redeem the same reset token twice.
	ok, err := s.takeResetToken(ctx, u.ID, token)
	if err != nil {
		return mapError(err)
	}
	if !ok {
		return verificationInvalid()
	}

	hash, err := s.passwords.Hash(newPassword)
	if err != nil {
		return mapError(err)
	}
	resetActor := actor
	resetActor.ID = u.ID
	resetActor.Username = u.Username
	resetActor.Email = u.Email
	resetActor.Nickname = u.Nickname
	if err := s.withPendingAudit(ctx, resetActor, ActionUserResetPassword, ResourceUser, idString(u.ID), map[string]any{
		"user_id":  idString(u.ID),
		"username": u.Username,
		"reason":   "password_reset",
	}, func() error {
		return s.users.SetUserPassword(ctx, u.ID, hash)
	}); err != nil {
		return err
	}
	s.revokeUserSessionsBestEffort(ctx, u.ID, "password reset")
	return nil
}

// storeResetToken persists the sha256 digest of a reset token with the
// configured TTL, failing closed on Redis errors.
func (s *Service) storeResetToken(ctx context.Context, userID int64, tokenHash string) error {
	return s.storeOneTimeToken(ctx, userResetKey(userID), tokenHash, time.Duration(s.resetTTL))
}

// takeResetToken atomically consumes a reset token (one-time use).
func (s *Service) takeResetToken(ctx context.Context, userID int64, token string) (bool, error) {
	return s.takeOneTimeToken(ctx, userResetKey(userID), token)
}

// sendResetEmail builds and sends the reset link email. SMTP failures are
// returned to the caller, which logs and swallows them for forgot-password.
func (s *Service) sendResetEmail(ctx context.Context, settings *userdomain.SystemSettings, toEmail, token string) error {
	link := s.frontendLink(settings.PublicFrontendURL, "/reset-password", toEmail, token)
	subject := fmt.Sprintf("%s: reset your password", settings.PlatformName)
	body := resetEmailBody(settings.PlatformName, link, s.resetTTL)
	return s.sendSystemEmail(ctx, settings, toEmail, subject, body)
}

func verificationInvalid() error {
	return apperr.New(400, apperr.CodeVerificationInvalid, "verification token invalid", apperr.ErrVerificationInvalid)
}

func resetEmailBody(platformName, link string, ttlNanos int64) string {
	hours := int64(1)
	if ttlNanos > 0 {
		hours = ttlNanos / int64(time.Hour)
		if hours < 1 {
			hours = 1
		}
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html><html><body style=\"font-family:sans-serif;max-width:560px;margin:0 auto\">")
	b.WriteString("<h2>")
	b.WriteString(escapeHTML(platformName))
	b.WriteString("</h2>")
	b.WriteString("<p>We received a request to reset your password. Choose a new password with the link below.</p>")
	b.WriteString("<p><a href=\"")
	b.WriteString(escapeHTMLAttr(link))
	b.WriteString("\">Reset my password</a></p>")
	b.WriteString("<p>This link is single-use and expires after ")
	b.WriteString(fmt.Sprintf("%d", hours))
	b.WriteString(" hour(s). If you did not request a password reset, you can safely ignore this message.</p>")
	b.WriteString("</body></html>")
	return b.String()
}
