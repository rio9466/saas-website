package usersvc

import (
	"context"

	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	userdomain "github.com/rio9466/easy-admin/server/internal/domain/user"
)

// AuthzPrincipal is live business-user authz state for transport middleware.
type AuthzPrincipal struct {
	UserID    int64
	Username  string
	Email     string
	Nickname  string
	Status    string
	AuthEpoch int64
}

// LoadUserAuthz loads live user enablement/status and auth epoch from primary
// storage so protected user endpoints never trust stale token state.
func (s *Service) LoadUserAuthz(ctx context.Context, userID int64) (*AuthzPrincipal, error) {
	if userID <= 0 {
		return nil, apperr.New(401, apperr.CodeUnauthorized, "unauthorized", apperr.ErrUnauthorized)
	}
	u, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return nil, mapError(err)
	}
	if u.Status == userdomain.StatusDisabled {
		return nil, apperr.New(403, apperr.CodeAccountDisabled, "account disabled", apperr.ErrAccountDisabled)
	}
	if u.Status == userdomain.StatusPendingVerification {
		return nil, apperr.New(403, apperr.CodeEmailNotVerified, "email not verified", apperr.ErrEmailNotVerified)
	}
	return &AuthzPrincipal{
		UserID:    u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Nickname:  u.Nickname,
		Status:    u.Status,
		AuthEpoch: u.AuthEpoch,
	}, nil
}
