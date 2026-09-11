package usersvc

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	userdomain "github.com/rio9466/easy-admin/server/internal/domain/user"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
)

// issueAndSendVerification mints a one-time token, stores its hash, and sends
// the email for a newly created pending account.
func (s *Service) issueAndSendVerification(ctx context.Context, meta Actor, settings *userdomain.SystemSettings, email string, userID int64) error {
	token, err := generateVerificationToken()
	if err != nil {
		return mapError(err)
	}
	if err := s.storeVerificationToken(ctx, userID, hashVerificationToken(token)); err != nil {
		return mapError(err)
	}
	if err := s.sendVerificationEmail(ctx, settings, email, token); err != nil {
		return mapError(err)
	}
	return nil
}

// VerifyEmail redeems a one-time verification token. The token is atomically
// consumed (single-use GETDEL), replay-safe (hash compare), and only ever
// activates the intended pending account. Success grants the registration
// starting points once via the deterministic ledger key.
func (s *Service) VerifyEmail(ctx context.Context, meta Actor, email, token string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	token = strings.TrimSpace(token)
	if email == "" || token == "" {
		return apperr.New(400, apperr.CodeVerificationInvalid, "verification token invalid", apperr.ErrVerificationInvalid)
	}
	if err := s.allowRate(ctx, rateKeyVerifyByIP(meta.SourceIP), verifyLimit, verifyWindow); err != nil {
		return err
	}

	u, err := s.users.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, primary.ErrNotFound) {
			// Generic response; the address is never revealed.
			return apperr.New(400, apperr.CodeVerificationInvalid, "verification token invalid", apperr.ErrVerificationInvalid)
		}
		return mapError(err)
	}
	if u.Status != userdomain.StatusPendingVerification {
		return apperr.New(400, apperr.CodeVerificationInvalid, "verification token invalid", apperr.ErrVerificationInvalid)
	}

	// Atomic one-time consumption BEFORE any account mutation. GETDEL means a
	// concurrent or replayed request can never redeem the same token twice.
	ok, err := s.takeVerificationToken(ctx, u.ID, token)
	if err != nil {
		return mapError(err)
	}
	if !ok {
		return apperr.New(400, apperr.CodeVerificationInvalid, "verification token invalid", apperr.ErrVerificationInvalid)
	}

	actor := meta
	actor.ID = u.ID
	actor.Username = u.Username
	actor.Email = u.Email
	actor.Nickname = u.Nickname

	now := time.Now().UTC()
	if err := s.recordSecurityEvent(ctx, actor, ActionUserVerifyEmail, ResourceEmailVerification, idString(u.ID), map[string]any{
		"username": u.Username,
		"user_id":  idString(u.ID),
	}, true); err != nil {
		return err
	}

	if err := s.users.SetUserStatus(ctx, u.ID, userdomain.StatusActive, &now); err != nil {
		return mapError(err)
	}

	// Grant registration starting points exactly once (deterministic key);
	// a replay of the grant is accepted as already applied.
	settings, err := s.users.GetSystemSettings(ctx)
	if err != nil {
		return nil
	}
	if !settings.RegistrationPoints.IsZero() {
		if err := s.grantRegistrationPoints(ctx, u.ID, settings.RegistrationPoints); err != nil {
			return err
		}
	}
	return nil
}

// ResendVerification invalidates the previous token, stores a fresh one, and
// emails it again. Rate-limited and audit-protected pending-first; only the
// intended pending account's token is ever replaced.
func (s *Service) ResendVerification(ctx context.Context, meta Actor, email string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return validationError("email is required")
	}
	if err := s.allowRate(ctx, rateKeyResend(meta.SourceIP), resendLimit, resendWindow); err != nil {
		return err
	}
	settings, err := s.users.GetSystemSettings(ctx)
	if err != nil {
		return mapError(err)
	}
	if !settings.EmailVerificationRequired {
		return apperr.New(400, apperr.CodeVerificationInvalid, "verification token invalid", apperr.ErrVerificationInvalid)
	}

	u, err := s.users.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, primary.ErrNotFound) {
			// Do not reveal whether the address exists.
			return apperr.New(400, apperr.CodeVerificationInvalid, "verification token invalid", apperr.ErrVerificationInvalid)
		}
		return mapError(err)
	}
	if u.Status != userdomain.StatusPendingVerification {
		return apperr.New(400, apperr.CodeVerificationInvalid, "verification token invalid", apperr.ErrVerificationInvalid)
	}

	actor := meta
	actor.ID = u.ID
	actor.Username = u.Username
	actor.Email = u.Email
	actor.Nickname = u.Nickname

	token, err := generateVerificationToken()
	if err != nil {
		return mapError(err)
	}
	// Audit-pending first: the token rotation write below must not run when
	// the durable audit insert fails.
	return s.withPendingAudit(ctx, actor, ActionUserResendVerification, ResourceEmailVerification, idString(u.ID), map[string]any{
		"user_id": idString(u.ID),
		"reason":  "resent_verification",
	}, func() error {
		if err := s.storeVerificationToken(ctx, u.ID, hashVerificationToken(token)); err != nil {
			return err
		}
		if err := s.sendVerificationEmail(ctx, settings, u.Email, token); err != nil {
			return err
		}
		return nil
	})
}
