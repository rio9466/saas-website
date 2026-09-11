package usersvc

import (
	"context"
	"net/url"
	"strings"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	userdomain "github.com/rio9466/easy-admin/server/internal/domain/user"
)

// UpdateProfileInput is the self-service profile update. Only nickname and
// avatar_url are bound; username, email, points, level, and status can never be
// changed through this operation.
type UpdateProfileInput struct {
	Nickname  *string
	AvatarURL *string
}

// UpdateProfile updates the authenticated user's own nickname and/or avatar
// under the pending-first audit policy.
func (s *Service) UpdateProfile(ctx context.Context, actor Actor, in UpdateProfileInput) (*userdomain.User, error) {
	if err := s.requireAuth(actor); err != nil {
		return nil, err
	}
	if in.Nickname == nil && in.AvatarURL == nil {
		return nil, validationError("at least one of nickname or avatar_url is required")
	}
	if in.Nickname != nil {
		nickname := strings.TrimSpace(*in.Nickname)
		if nickname == "" {
			return nil, validationError("nickname must not be empty")
		}
		in.Nickname = &nickname
	}
	if in.AvatarURL != nil {
		avatar := strings.TrimSpace(*in.AvatarURL)
		if err := validateAvatarURL(avatar); err != nil {
			return nil, err
		}
		in.AvatarURL = &avatar
	}

	details := map[string]any{"user_id": idString(actor.ID)}
	if in.Nickname != nil {
		details["nickname"] = *in.Nickname
	}
	if in.AvatarURL != nil {
		details["avatar_url"] = *in.AvatarURL
	}

	var saved *userdomain.User
	err := s.withPendingAudit(ctx, actor, ActionUserUpdate, ResourceUser, idString(actor.ID), details, func() error {
		if err := s.users.UpdateUserProfile(ctx, actor.ID, nil, in.Nickname, in.AvatarURL, nil); err != nil {
			return err
		}
		u, err := s.users.GetUserByID(ctx, actor.ID)
		if err != nil {
			return err
		}
		saved = u
		return nil
	})
	if err != nil {
		return nil, err
	}
	return saved, nil
}

// ChangePassword verifies the current password, stores a new hash, and revokes
// every session the user holds (auth_epoch bump + Redis cleanup). The audit
// write is pending-first: when it is unavailable the password is not changed.
func (s *Service) ChangePassword(ctx context.Context, actor Actor, currentPassword, newPassword string) error {
	if err := s.requireAuth(actor); err != nil {
		return err
	}
	if strings.TrimSpace(currentPassword) == "" || strings.TrimSpace(newPassword) == "" {
		return validationError("current and new password are required")
	}
	if len(newPassword) < 8 || len(newPassword) > 72 {
		return validationError("password must be 8-72 bytes")
	}

	u, err := s.users.GetUserByUsername(ctx, actor.Username)
	if err != nil {
		return mapError(err)
	}
	if err := s.passwords.Compare(u.PasswordHash, currentPassword); err != nil {
		return apperr.New(400, apperr.CodeInvalidPassword, "invalid password", apperr.ErrInvalidPassword)
	}
	hash, err := s.passwords.Hash(newPassword)
	if err != nil {
		return mapError(err)
	}
	if err := s.withPendingAudit(ctx, actor, ActionUserChangePassword, ResourceUser, idString(actor.ID), map[string]any{
		"user_id":  idString(actor.ID),
		"username": u.Username,
	}, func() error {
		return s.users.SetUserPassword(ctx, actor.ID, hash)
	}); err != nil {
		return err
	}
	s.revokeUserSessionsBestEffort(ctx, actor.ID, "password change")
	return nil
}

// ListMyPointTransactions returns the authenticated user's own ledger page
// (newest first, paginated).
func (s *Service) ListMyPointTransactions(ctx context.Context, actor Actor, page, pageSize int) (adminauth.Page[userdomain.PointTransaction], error) {
	if err := s.requireAuth(actor); err != nil {
		return adminauth.Page[userdomain.PointTransaction]{}, err
	}
	out, err := s.users.ListPointTransactions(ctx, actor.ID, page, pageSize)
	if err != nil {
		return adminauth.Page[userdomain.PointTransaction]{}, mapError(err)
	}
	return out, nil
}

// validateAvatarURL allows an empty value (clear), an absolute http/https URL,
// or a root-relative path. Other schemes (javascript:, data:, ...) and
// protocol-relative URLs are rejected so a stored avatar URL cannot be abused.
func validateAvatarURL(raw string) error {
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return validationError("invalid avatar_url")
	}
	switch {
	case u.Scheme == "http" || u.Scheme == "https":
		if u.Host == "" {
			return validationError("invalid avatar_url")
		}
		return nil
	case u.Scheme == "" && u.Host == "" && strings.HasPrefix(raw, "/"):
		return nil
	default:
		return validationError("invalid avatar_url")
	}
}
