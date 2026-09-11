package usersvc

import (
	"context"
	"strings"
	"time"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	userdomain "github.com/rio9466/easy-admin/server/internal/domain/user"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
)

// CreateUserInput for administrator-created accounts. Only the approved fields
// are accepted; all other values use validated system defaults.
type CreateUserInput struct {
	Username string
	Email    string
	Password string
	Nickname string
}

// UpdateUserInput updates only the profile fields the generic update endpoint
// owns. Username, registration IP/time, points, level mode, status, and
// password have dedicated operations and can never be changed here.
type UpdateUserInput struct {
	Email     *string
	Nickname  *string
	AvatarURL *string
	Remark    *string
}

// UserListFilter filters the business-user page.
type UserListFilter struct {
	Query   string
	Status  string
	LevelID *int64
}

// ListUsers returns a filtered page of business users.
func (s *Service) ListUsers(ctx context.Context, actor Actor, page, pageSize int, filter UserListFilter) (adminauth.Page[userdomain.User], error) {
	if err := s.requirePermission(ctx, actor, permissionCustomerRead); err != nil {
		return adminauth.Page[userdomain.User]{}, err
	}
	out, err := s.users.ListUsers(ctx, page, pageSize, primary.UserListFilter{
		Query:   filter.Query,
		Status:  filter.Status,
		LevelID: filter.LevelID,
	})
	if err != nil {
		return adminauth.Page[userdomain.User]{}, mapError(err)
	}
	return out, nil
}

// GetUser returns one business user by id.
func (s *Service) GetUser(ctx context.Context, actor Actor, id int64) (*userdomain.User, error) {
	if err := s.requirePermission(ctx, actor, permissionCustomerRead); err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, validationError("invalid user id")
	}
	u, err := s.users.GetUserByID(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	return u, nil
}

// CreateUser creates an active, email-verified business user with optional
// nickname; defaults come from validated system settings.
func (s *Service) CreateUser(ctx context.Context, actor Actor, in CreateUserInput) (*userdomain.User, error) {
	if err := s.requirePermission(ctx, actor, permissionCustomerCreate); err != nil {
		return nil, err
	}
	username := strings.TrimSpace(in.Username)
	email := strings.TrimSpace(strings.ToLower(in.Email))
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

	settings, err := s.users.GetSystemSettings(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	if settings.DefaultLevelID <= 0 {
		return nil, apperr.New(503, apperr.CodeDependencyUnavailable, "dependency unavailable", apperr.ErrDependencyUnavailable)
	}

	now := time.Now().UTC()
	nickname := strings.TrimSpace(in.Nickname)
	if nickname == "" {
		nickname = username
	}
	nu := &userdomain.User{
		Username:        username,
		Email:           email,
		PasswordHash:    hash,
		AvatarURL:       settings.DefaultAvatarURL,
		Nickname:        nickname,
		RegistrationIP:  actor.SourceIP,
		Status:          userdomain.StatusActive,
		EmailVerifiedAt: &now,
		LevelMode:       userdomain.LevelModeAuto,
	}

	newID, err := s.createAudited(ctx, actor, ActionUserCreate, ResourceUser, map[string]any{
		"username": username,
		"email":    email,
		"nickname": nickname,
	}, func() (int64, error) {
		if err := s.users.CreateUser(ctx, nu, settings.DefaultLevelID); err != nil {
			return 0, err
		}
		return nu.ID, nil
	})
	if err != nil {
		return nil, err
	}
	// Reload so the response carries the persisted level and generated defaults.
	saved, err := s.users.GetUserByID(ctx, newID)
	if err != nil {
		return nil, mapError(err)
	}
	return saved, nil
}

// UpdateUser updates email, nickname, avatar URL, and remark only.
func (s *Service) UpdateUser(ctx context.Context, actor Actor, id int64, in UpdateUserInput) (*userdomain.User, error) {
	if err := s.requirePermission(ctx, actor, permissionCustomerUpdate); err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, validationError("invalid user id")
	}
	if in.Email != nil {
		email := strings.TrimSpace(strings.ToLower(*in.Email))
		if err := validateEmail(email); err != nil {
			return nil, err
		}
		in.Email = &email
	}
	if in.Nickname != nil && strings.TrimSpace(*in.Nickname) == "" {
		return nil, validationError("nickname must not be empty")
	}
	if in.AvatarURL != nil {
		url := strings.TrimSpace(*in.AvatarURL)
		in.AvatarURL = &url
	}
	remark := ""
	if in.Remark != nil {
		remark = *in.Remark
		in.Remark = &remark
	}

	var saved *userdomain.User
	err := s.withPendingAudit(ctx, actor, ActionUserUpdate, ResourceUser, idString(id), map[string]any{
		"user_id": idString(id),
	}, func() error {
		if err := s.users.UpdateUserProfile(ctx, id, in.Email, in.Nickname, in.AvatarURL, in.Remark); err != nil {
			return err
		}
		u, err := s.users.GetUserByID(ctx, id)
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

// EnableUser re-enables a disabled business user.
func (s *Service) EnableUser(ctx context.Context, actor Actor, id int64) (*userdomain.User, error) {
	if err := s.requirePermission(ctx, actor, permissionCustomerDisable); err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, validationError("invalid user id")
	}
	current, err := s.users.GetUserByID(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	if current.Status == userdomain.StatusPendingVerification {
		return nil, apperr.New(400, apperr.CodeVerificationInvalid, "verification token invalid", apperr.ErrVerificationInvalid)
	}
	if current.Status == userdomain.StatusActive {
		return current, nil
	}
	var saved *userdomain.User
	err = s.withPendingAudit(ctx, actor, ActionUserEnable, ResourceUser, idString(id), map[string]any{
		"user_id":    idString(id),
		"new_status": userdomain.StatusActive,
		"old_status": current.Status,
	}, func() error {
		if err := s.users.SetUserEnabled(ctx, id, true); err != nil {
			return err
		}
		u, err := s.users.GetUserByID(ctx, id)
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

// DisableUser disables an active business user and invalidates its sessions
// via the auth_epoch bump.
func (s *Service) DisableUser(ctx context.Context, actor Actor, id int64) (*userdomain.User, error) {
	if err := s.requirePermission(ctx, actor, permissionCustomerDisable); err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, validationError("invalid user id")
	}
	current, err := s.users.GetUserByID(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	if current.Status == userdomain.StatusDisabled {
		return current, nil
	}
	var saved *userdomain.User
	err = s.withPendingAudit(ctx, actor, ActionUserDisable, ResourceUser, idString(id), map[string]any{
		"user_id":    idString(id),
		"new_status": userdomain.StatusDisabled,
		"old_status": current.Status,
	}, func() error {
		if err := s.users.SetUserEnabled(ctx, id, false); err != nil {
			return err
		}
		u, err := s.users.GetUserByID(ctx, id)
		if err != nil {
			return err
		}
		saved = u
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.revokeUserSessionsBestEffort(ctx, id, "user disabled")
	return saved, nil
}

// ResetUserPassword sets a new password and invalidates every session.
func (s *Service) ResetUserPassword(ctx context.Context, actor Actor, id int64, newPassword string) error {
	if err := s.requirePermission(ctx, actor, permissionCustomerResetPwd); err != nil {
		return err
	}
	if id <= 0 {
		return validationError("invalid user id")
	}
	if len(newPassword) < 8 || len(newPassword) > 72 {
		return apperr.New(400, apperr.CodeInvalidPassword, "invalid password", apperr.ErrInvalidPassword)
	}
	hash, err := s.passwords.Hash(newPassword)
	if err != nil {
		return mapError(err)
	}
	if err := s.withPendingAudit(ctx, actor, ActionUserResetPassword, ResourceUser, idString(id), map[string]any{
		"user_id": idString(id),
	}, func() error {
		return s.users.SetUserPassword(ctx, id, hash)
	}); err != nil {
		return err
	}
	s.revokeUserSessionsBestEffort(ctx, id, "password reset")
	return nil
}

func (s *Service) revokeUserSessionsBestEffort(ctx context.Context, userID int64, reason string) {
	if err := s.sessions.RevokeAllForAdmin(ctx, userID); err != nil {
		s.logger.Error("user session revocation cleanup failed; auth epoch invalidates old sessions",
			"user_id", idString(userID),
			"reason", reason,
		)
	}
}
