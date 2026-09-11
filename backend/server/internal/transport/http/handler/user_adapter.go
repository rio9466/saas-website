package handler

import (
	"context"
	"time"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	userdomain "github.com/rio9466/easy-admin/server/internal/domain/user"
	usersvc "github.com/rio9466/easy-admin/server/internal/service/usersvc"
	"github.com/rio9466/easy-admin/server/internal/transport/http/middleware"
)

// UserServiceAdapter adapts *usersvc.Service to the transport interfaces for
// the public/user and admin-managed business-user surfaces.
type UserServiceAdapter struct {
	Svc *usersvc.Service
}

var (
	_ UserClientService = (*UserServiceAdapter)(nil)
	_ UserAdminService  = (*UserServiceAdapter)(nil)
)

func (a *UserServiceAdapter) LoadUserAuthz(ctx context.Context, userID int64) (*middleware.UserAuthzPrincipal, error) {
	p, err := a.Svc.LoadUserAuthz(ctx, userID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, nil
	}
	return &middleware.UserAuthzPrincipal{
		UserID:    p.UserID,
		Username:  p.Username,
		Email:     p.Email,
		Nickname:  p.Nickname,
		Status:    p.Status,
		AuthEpoch: p.AuthEpoch,
	}, nil
}

// toSvcClientActor maps a request actor that authenticated as a business user.
func toSvcClientActor(a Actor) usersvc.Actor {
	return usersvc.Actor{
		ID:        a.ID,
		Username:  a.Username,
		Email:     a.Email,
		Nickname:  a.Nickname,
		SessionID: a.SessionID,
		RequestID: a.RequestID,
		SourceIP:  a.SourceIP,
		UserAgent: a.UserAgent,
	}
}

// toSvcAdminActor maps a request actor that authenticated as a backend
// administrator (admin-facing business-user operations).
func toSvcAdminActor(a Actor) usersvc.Actor {
	return usersvc.Actor{
		AdminID:          a.ID,
		AdminUsername:    a.Username,
		AdminDisplayName: a.DisplayName,
		AdminRoleCodes:   a.RoleCodes,
		RequestID:        a.RequestID,
		SourceIP:         a.SourceIP,
		UserAgent:        a.UserAgent,
	}
}

func (a *UserServiceAdapter) GetPublicSettings(ctx context.Context) (*PublicSettingsData, error) {
	ps, err := a.Svc.GetPublicSettings(ctx)
	if err != nil {
		return nil, err
	}
	return &PublicSettingsData{
		PlatformName:              ps.PlatformName,
		PublicFrontendURL:         ps.PublicFrontendURL,
		PublicAPIURL:              ps.PublicAPIURL,
		RegistrationEnabled:       ps.RegistrationEnabled,
		UsernameLoginEnabled:      ps.UsernameLoginEnabled,
		EmailLoginEnabled:         ps.EmailLoginEnabled,
		EmailVerificationRequired: ps.EmailVerificationRequired,
		DefaultAvatarURL:          ps.DefaultAvatarURL,
	}, nil
}

func (a *UserServiceAdapter) Register(ctx context.Context, actor Actor, in UserRegisterInput) (*userdomain.User, error) {
	return a.Svc.Register(ctx, toSvcClientActor(actor), usersvc.RegisterInput{
		Username: in.Username,
		Email:    in.Email,
		Password: in.Password,
		SourceIP: actor.SourceIP,
	})
}

func (a *UserServiceAdapter) VerifyEmail(ctx context.Context, actor Actor, email, token string) error {
	return a.Svc.VerifyEmail(ctx, toSvcClientActor(actor), email, token)
}

func (a *UserServiceAdapter) ResendVerification(ctx context.Context, actor Actor, email string) error {
	return a.Svc.ResendVerification(ctx, toSvcClientActor(actor), email)
}

func (a *UserServiceAdapter) Login(ctx context.Context, actor Actor, identifier, password, sourceIP string) (*UserLoginResult, *userdomain.User, error) {
	session, me, err := a.Svc.Login(ctx, toSvcClientActor(actor), identifier, password, sourceIP)
	if err != nil {
		return nil, nil, err
	}
	return mapUserLoginResult(session), me, nil
}

func (a *UserServiceAdapter) Refresh(ctx context.Context, actor Actor, refreshToken string) (*UserLoginResult, *userdomain.User, error) {
	session, me, err := a.Svc.Refresh(ctx, toSvcClientActor(actor), refreshToken)
	if err != nil {
		return nil, nil, err
	}
	return mapUserLoginResult(session), me, nil
}

func (a *UserServiceAdapter) Logout(ctx context.Context, actor Actor) error {
	return a.Svc.Logout(ctx, toSvcClientActor(actor))
}

func (a *UserServiceAdapter) Me(ctx context.Context, actor Actor, userID int64) (*userdomain.User, error) {
	return a.Svc.Me(ctx, toSvcClientActor(actor), userID)
}

func mapUserLoginResult(res *usersvc.SessionResult) *UserLoginResult {
	if res == nil {
		return nil
	}
	return &UserLoginResult{
		AccessToken:      res.AccessToken,
		RefreshToken:     res.RefreshToken,
		RefreshExpiresAt: res.RefreshExpiresAt,
		ExpiresIn:        int64((15 * time.Minute) / time.Second),
	}
}

func (a *UserServiceAdapter) ListUsers(ctx context.Context, actor Actor, page, pageSize int, filter UserListInput) (adminauth.Page[userdomain.User], error) {
	return a.Svc.ListUsers(ctx, toSvcAdminActor(actor), page, pageSize, usersvc.UserListFilter{
		Query:   filter.Query,
		Status:  filter.Status,
		LevelID: filter.LevelID,
	})
}

func (a *UserServiceAdapter) GetUser(ctx context.Context, actor Actor, id int64) (*userdomain.User, error) {
	return a.Svc.GetUser(ctx, toSvcAdminActor(actor), id)
}

func (a *UserServiceAdapter) CreateUser(ctx context.Context, actor Actor, in CreateUserInput) (*userdomain.User, error) {
	nick := ""
	if in.Nickname != nil {
		nick = *in.Nickname
	}
	return a.Svc.CreateUser(ctx, toSvcAdminActor(actor), usersvc.CreateUserInput{
		Username: in.Username,
		Email:    in.Email,
		Password: in.Password,
		Nickname: nick,
	})
}

func (a *UserServiceAdapter) UpdateUser(ctx context.Context, actor Actor, id int64, in UpdateUserInput) (*userdomain.User, error) {
	return a.Svc.UpdateUser(ctx, toSvcAdminActor(actor), id, usersvc.UpdateUserInput{
		Email:     in.Email,
		Nickname:  in.Nickname,
		AvatarURL: in.AvatarURL,
		Remark:    in.Remark,
	})
}

func (a *UserServiceAdapter) EnableUser(ctx context.Context, actor Actor, id int64) (*userdomain.User, error) {
	return a.Svc.EnableUser(ctx, toSvcAdminActor(actor), id)
}

func (a *UserServiceAdapter) DisableUser(ctx context.Context, actor Actor, id int64) (*userdomain.User, error) {
	return a.Svc.DisableUser(ctx, toSvcAdminActor(actor), id)
}

func (a *UserServiceAdapter) ResetUserPassword(ctx context.Context, actor Actor, id int64, newPassword string) error {
	return a.Svc.ResetUserPassword(ctx, toSvcAdminActor(actor), id, newPassword)
}

func (a *UserServiceAdapter) AdjustPoints(ctx context.Context, actor Actor, userID int64, in AdjustPointsInput) (*userdomain.PointTransaction, error) {
	return a.Svc.AdjustPoints(ctx, toSvcAdminActor(actor), usersvc.AdjustPointsInput{
		UserID:           userID,
		PointsDelta:      in.PointsDelta,
		ConsumptionDelta: in.ConsumptionDelta,
		Reason:           in.Reason,
		IdempotencyKey:   in.IdempotencyKey,
	})
}

func (a *UserServiceAdapter) ListPointTransactions(ctx context.Context, actor Actor, userID int64, page, pageSize int) (adminauth.Page[userdomain.PointTransaction], error) {
	return a.Svc.ListPointTransactions(ctx, toSvcAdminActor(actor), userID, page, pageSize)
}

func (a *UserServiceAdapter) AssignUserLevel(ctx context.Context, actor Actor, in AssignLevelInput) (*userdomain.User, error) {
	return a.Svc.AssignUserLevel(ctx, toSvcAdminActor(actor), usersvc.AssignUserLevelInput{
		UserID:  in.UserID,
		LevelID: in.LevelID,
		Mode:    in.Mode,
	})
}

func (a *UserServiceAdapter) ListUserLevels(ctx context.Context, actor Actor) ([]userdomain.UserLevel, error) {
	return a.Svc.ListUserLevels(ctx, toSvcAdminActor(actor))
}

func (a *UserServiceAdapter) GetUserLevel(ctx context.Context, actor Actor, id int64) (*userdomain.UserLevel, error) {
	return a.Svc.GetUserLevel(ctx, toSvcAdminActor(actor), id)
}

func (a *UserServiceAdapter) CreateUserLevel(ctx context.Context, actor Actor, in CreateLevelInput) (*userdomain.UserLevel, error) {
	return a.Svc.CreateUserLevel(ctx, toSvcAdminActor(actor), usersvc.CreateUserLevelInput{
		Code:            in.Code,
		Name:            in.Name,
		IconURL:         in.IconURL,
		ThresholdPoints: in.ThresholdPoints,
		SortOrder:       in.SortOrder,
		Enabled:         in.Enabled,
	})
}

func (a *UserServiceAdapter) UpdateUserLevel(ctx context.Context, actor Actor, id int64, in UpdateLevelInput) (*userdomain.UserLevel, error) {
	return a.Svc.UpdateUserLevel(ctx, toSvcAdminActor(actor), id, usersvc.UpdateUserLevelInput{
		Name:            in.Name,
		IconURL:         in.IconURL,
		ThresholdPoints: in.ThresholdPoints,
		SortOrder:       in.SortOrder,
		Enabled:         in.Enabled,
	})
}

func (a *UserServiceAdapter) GetSystemSettings(ctx context.Context, actor Actor) (*userdomain.SystemSettings, error) {
	return a.Svc.GetSystemSettingsForActor(ctx, toSvcAdminActor(actor))
}

func (a *UserServiceAdapter) UpdateSystemSettings(ctx context.Context, actor Actor, in UpdateSystemSettingsInput) (*userdomain.SystemSettings, error) {
	return a.Svc.UpdateSystemSettings(ctx, toSvcAdminActor(actor), usersvc.UpdateSystemSettingsInput{
		PlatformName:              in.PlatformName,
		PublicFrontendURL:         in.PublicFrontendURL,
		PublicAPIURL:              in.PublicAPIURL,
		RegistrationEnabled:       in.RegistrationEnabled,
		UsernameLoginEnabled:      in.UsernameLoginEnabled,
		EmailLoginEnabled:         in.EmailLoginEnabled,
		EmailVerificationRequired: in.EmailVerificationRequired,
		DefaultLevelID:            in.DefaultLevelID,
		DefaultAvatarURL:          in.DefaultAvatarURL,
		RegistrationPoints:        in.RegistrationPoints,
		SMTPEnabled:               in.SMTPEnabled,
		SMTPHost:                  in.SMTPHost,
		SMTPPort:                  in.SMTPPort,
		SMTPUsername:              in.SMTPUsername,
		SMTPPassword:              in.SMTPPassword,
		SMTPFromEmail:             in.SMTPFromEmail,
		SMTPFromName:              in.SMTPFromName,
		SMTPTLSMode:               in.SMTPTLSMode,
		Version:                   in.Version,
	})
}
