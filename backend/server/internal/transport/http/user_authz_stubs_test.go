package http_test

import (
	"context"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	userdomain "github.com/rio9466/easy-admin/server/internal/domain/user"
	"github.com/rio9466/easy-admin/server/internal/transport/http/handler"
	"github.com/rio9466/easy-admin/server/internal/transport/http/middleware"
)

// stubUserClient implements handler.UserClientService for router parity and
// authz routing tests without touching real stores.
type stubUserClient struct {
	principal *middleware.UserAuthzPrincipal
}

func (s *stubUserClient) LoadUserAuthz(context.Context, int64) (*middleware.UserAuthzPrincipal, error) {
	if s.principal == nil {
		return &middleware.UserAuthzPrincipal{UserID: 1, Status: "active", AuthEpoch: 1}, nil
	}
	return s.principal, nil
}

func (s *stubUserClient) GetPublicSettings(context.Context) (*handler.PublicSettingsData, error) {
	return &handler.PublicSettingsData{PlatformName: "easy-admin"}, nil
}

func (s *stubUserClient) Register(context.Context, handler.Actor, handler.UserRegisterInput) (*userdomain.User, error) {
	return &userdomain.User{ID: 1, Username: "stub", Status: userdomain.StatusActive}, nil
}

func (s *stubUserClient) VerifyEmail(context.Context, handler.Actor, string, string) error {
	return nil
}
func (s *stubUserClient) ResendVerification(context.Context, handler.Actor, string) error { return nil }

func (s *stubUserClient) Login(context.Context, handler.Actor, string, string, string) (*handler.UserLoginResult, *userdomain.User, error) {
	return &handler.UserLoginResult{AccessToken: "stub-access", RefreshToken: "stub-refresh"}, &userdomain.User{ID: 1}, nil
}

func (s *stubUserClient) Refresh(context.Context, handler.Actor, string) (*handler.UserLoginResult, *userdomain.User, error) {
	return &handler.UserLoginResult{AccessToken: "stub-access", RefreshToken: "stub-refresh"}, &userdomain.User{ID: 1}, nil
}

func (s *stubUserClient) Logout(context.Context, handler.Actor) error { return nil }

func (s *stubUserClient) Me(context.Context, handler.Actor, int64) (*userdomain.User, error) {
	return &userdomain.User{ID: 1, Username: "stub", Status: userdomain.StatusActive}, nil
}

// stubUserAdmin implements handler.UserAdminService for routing tests.
type stubUserAdmin struct{}

func (s *stubUserAdmin) ListUsers(context.Context, handler.Actor, int, int, handler.UserListInput) (adminauth.Page[userdomain.User], error) {
	return adminauth.Page[userdomain.User]{Items: []userdomain.User{}, Total: 0, Page: 1, PageSize: 20}, nil
}

func (s *stubUserAdmin) GetUser(context.Context, handler.Actor, int64) (*userdomain.User, error) {
	return &userdomain.User{ID: 1}, nil
}

func (s *stubUserAdmin) CreateUser(context.Context, handler.Actor, handler.CreateUserInput) (*userdomain.User, error) {
	return &userdomain.User{ID: 1}, nil
}

func (s *stubUserAdmin) UpdateUser(context.Context, handler.Actor, int64, handler.UpdateUserInput) (*userdomain.User, error) {
	return &userdomain.User{ID: 1}, nil
}

func (s *stubUserAdmin) EnableUser(context.Context, handler.Actor, int64) (*userdomain.User, error) {
	return &userdomain.User{ID: 1}, nil
}

func (s *stubUserAdmin) DisableUser(context.Context, handler.Actor, int64) (*userdomain.User, error) {
	return &userdomain.User{ID: 1}, nil
}

func (s *stubUserAdmin) ResetUserPassword(context.Context, handler.Actor, int64, string) error {
	return nil
}

func (s *stubUserAdmin) AdjustPoints(context.Context, handler.Actor, int64, handler.AdjustPointsInput) (*userdomain.PointTransaction, error) {
	return &userdomain.PointTransaction{ID: 1}, nil
}

func (s *stubUserAdmin) ListPointTransactions(context.Context, handler.Actor, int64, int, int) (adminauth.Page[userdomain.PointTransaction], error) {
	return adminauth.Page[userdomain.PointTransaction]{Items: []userdomain.PointTransaction{}}, nil
}

func (s *stubUserAdmin) AssignUserLevel(context.Context, handler.Actor, handler.AssignLevelInput) (*userdomain.User, error) {
	return &userdomain.User{ID: 1}, nil
}

func (s *stubUserAdmin) ListUserLevels(context.Context, handler.Actor) ([]userdomain.UserLevel, error) {
	return []userdomain.UserLevel{}, nil
}

func (s *stubUserAdmin) GetUserLevel(context.Context, handler.Actor, int64) (*userdomain.UserLevel, error) {
	return &userdomain.UserLevel{ID: 1}, nil
}

func (s *stubUserAdmin) CreateUserLevel(context.Context, handler.Actor, handler.CreateLevelInput) (*userdomain.UserLevel, error) {
	return &userdomain.UserLevel{ID: 1}, nil
}

func (s *stubUserAdmin) UpdateUserLevel(context.Context, handler.Actor, int64, handler.UpdateLevelInput) (*userdomain.UserLevel, error) {
	return &userdomain.UserLevel{ID: 1}, nil
}

func (s *stubUserAdmin) GetSystemSettings(context.Context, handler.Actor) (*userdomain.SystemSettings, error) {
	return &userdomain.SystemSettings{PlatformName: "easy-admin", Version: 1}, nil
}

func (s *stubUserAdmin) UpdateSystemSettings(context.Context, handler.Actor, handler.UpdateSystemSettingsInput) (*userdomain.SystemSettings, error) {
	return &userdomain.SystemSettings{PlatformName: "easy-admin", Version: 2}, nil
}
