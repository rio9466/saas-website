package handler

import (
	"context"
	"time"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/repository/logdb"
	adminsvc "github.com/rio9466/easy-admin/server/internal/service/adminauth"
	"github.com/rio9466/easy-admin/server/internal/transport/http/middleware"
)

// ServiceAdapter adapts *adminsvc.Service to AdminAuthService.
type ServiceAdapter struct {
	Svc *adminsvc.Service
}

var _ AdminAuthService = (*ServiceAdapter)(nil)

func (a *ServiceAdapter) LoadAuthz(ctx context.Context, adminID int64) (*middleware.AuthzPrincipal, error) {
	p, err := a.Svc.LoadAuthz(ctx, adminID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, nil
	}
	return &middleware.AuthzPrincipal{
		AdminID:         p.AdminID,
		Username:        p.Username,
		DisplayName:     p.DisplayName,
		Enabled:         p.Enabled,
		AuthEpoch:       p.AuthEpoch,
		RoleCodes:       p.RoleCodes,
		PermissionCodes: p.PermissionCodes,
	}, nil
}

func (a *ServiceAdapter) Login(ctx context.Context, username, password string, actor Actor) (*LoginResult, error) {
	res, err := a.Svc.Login(ctx, toSvcActor(actor), username, password)
	return mapLoginResult(res), err
}

func (a *ServiceAdapter) Refresh(ctx context.Context, refreshToken string, actor Actor) (*LoginResult, error) {
	res, err := a.Svc.Refresh(ctx, toSvcActor(actor), refreshToken)
	return mapLoginResult(res), err
}

func (a *ServiceAdapter) Logout(ctx context.Context, actor Actor) error {
	return a.Svc.Logout(ctx, toSvcActor(actor))
}

func (a *ServiceAdapter) Me(ctx context.Context, actor Actor) (*MeResult, error) {
	res, err := a.Svc.Me(ctx, toSvcActor(actor))
	if err != nil {
		return nil, err
	}
	return mapMeResult(res), nil
}

func (a *ServiceAdapter) ChangePassword(ctx context.Context, actor Actor, currentPassword, newPassword string) error {
	return a.Svc.ChangePassword(ctx, toSvcActor(actor), currentPassword, newPassword)
}

func (a *ServiceAdapter) UpdateMyProfile(ctx context.Context, actor Actor, displayName string) (*MeResult, error) {
	res, err := a.Svc.UpdateMyProfile(ctx, toSvcActor(actor), displayName)
	if err != nil {
		return nil, err
	}
	return mapMeResult(res), nil
}

func (a *ServiceAdapter) ListAdministrators(ctx context.Context, actor Actor, page, pageSize int, query string) (adminauth.Page[adminauth.Administrator], error) {
	return a.Svc.ListAdministrators(ctx, toSvcActor(actor), page, pageSize, query)
}

func (a *ServiceAdapter) GetAdministrator(ctx context.Context, actor Actor, id int64) (*adminauth.Administrator, error) {
	return a.Svc.GetAdministrator(ctx, toSvcActor(actor), id)
}

func (a *ServiceAdapter) CreateAdministrator(ctx context.Context, actor Actor, in CreateAdministratorInput) (*adminauth.Administrator, error) {
	return a.Svc.CreateAdministrator(ctx, toSvcActor(actor), adminsvc.CreateAdministratorInput{
		Username:    in.Username,
		Password:    in.Password,
		DisplayName: in.DisplayName,
		Enabled:     true,
		RoleCodes:   in.RoleCodes,
	})
}

func (a *ServiceAdapter) UpdateAdministrator(ctx context.Context, actor Actor, id int64, in UpdateAdministratorInput) (*adminauth.Administrator, error) {
	return a.Svc.UpdateAdministrator(ctx, toSvcActor(actor), id, adminsvc.UpdateAdministratorInput{
		DisplayName: in.DisplayName,
	})
}

func (a *ServiceAdapter) EnableAdministrator(ctx context.Context, actor Actor, id int64) (*adminauth.Administrator, error) {
	return a.Svc.EnableAdministrator(ctx, toSvcActor(actor), id)
}

func (a *ServiceAdapter) DisableAdministrator(ctx context.Context, actor Actor, id int64) (*adminauth.Administrator, error) {
	return a.Svc.DisableAdministrator(ctx, toSvcActor(actor), id)
}

func (a *ServiceAdapter) ResetAdministratorPassword(ctx context.Context, actor Actor, id int64, newPassword string) error {
	return a.Svc.ResetAdministratorPassword(ctx, toSvcActor(actor), id, newPassword)
}

func (a *ServiceAdapter) AssignAdministratorRoles(ctx context.Context, actor Actor, id int64, roleCodes []string) (*adminauth.Administrator, error) {
	return a.Svc.AssignAdministratorRoles(ctx, toSvcActor(actor), id, roleCodes)
}

func (a *ServiceAdapter) ListRoles(ctx context.Context, actor Actor) ([]adminauth.Role, error) {
	return a.Svc.ListRoles(ctx, toSvcActor(actor))
}

func (a *ServiceAdapter) GetRole(ctx context.Context, actor Actor, id int64) (*adminauth.Role, error) {
	return a.Svc.GetRole(ctx, toSvcActor(actor), id)
}

func (a *ServiceAdapter) CreateRole(ctx context.Context, actor Actor, in CreateRoleInput) (*adminauth.Role, error) {
	return a.Svc.CreateRole(ctx, toSvcActor(actor), adminsvc.CreateRoleInput{
		Code:        in.Code,
		Name:        in.Name,
		Description: in.Description,
	})
}

func (a *ServiceAdapter) UpdateRole(ctx context.Context, actor Actor, id int64, in UpdateRoleInput) (*adminauth.Role, error) {
	return a.Svc.UpdateRole(ctx, toSvcActor(actor), id, adminsvc.UpdateRoleInput{
		Name:        in.Name,
		Description: in.Description,
		Enabled:     in.Enabled,
	})
}

func (a *ServiceAdapter) ReplaceRolePermissions(ctx context.Context, actor Actor, id int64, permissionCodes []string) (*adminauth.Role, error) {
	return a.Svc.ReplaceRolePermissions(ctx, toSvcActor(actor), id, permissionCodes)
}

func (a *ServiceAdapter) ListPermissions(ctx context.Context, actor Actor) ([]adminauth.Permission, error) {
	return a.Svc.ListPermissions(ctx, toSvcActor(actor))
}

func (a *ServiceAdapter) ListAuditEvents(ctx context.Context, actor Actor, in AuditListInput) (adminauth.Page[adminauth.AuditEvent], error) {
	return a.Svc.ListAuditEvents(ctx, toSvcActor(actor), logdb.AuditListFilter{
		ActorID:      in.ActorID,
		Action:       in.Action,
		ResourceType: in.ResourceType,
		Outcome:      in.Outcome,
		RequestID:    in.RequestID,
		From:         in.From,
		To:           in.To,
		Page:         in.Page,
		PageSize:     in.PageSize,
	})
}

func (a *ServiceAdapter) GetAuditEvent(ctx context.Context, actor Actor, id int64) (*adminauth.AuditEvent, error) {
	return a.Svc.GetAuditEvent(ctx, toSvcActor(actor), id)
}

func toSvcActor(a Actor) adminsvc.Actor {
	return adminsvc.Actor{
		ID:          a.ID,
		Username:    a.Username,
		DisplayName: a.DisplayName,
		RoleCodes:   a.RoleCodes,
		SessionID:   a.SessionID,
		RequestID:   a.RequestID,
		SourceIP:    a.SourceIP,
		UserAgent:   a.UserAgent,
	}
}

func mapLoginResult(res *adminsvc.LoginResult) *LoginResult {
	if res == nil {
		return nil
	}
	expiresIn := int64((15 * time.Minute) / time.Second)
	if !res.RefreshExpiresAt.IsZero() {
		// Access token lifetime is product-fixed at 15 minutes.
		expiresIn = int64((15 * time.Minute) / time.Second)
	}
	return &LoginResult{
		AccessToken:      res.AccessToken,
		RefreshToken:     res.RefreshToken,
		ExpiresIn:        expiresIn,
		RefreshExpiresAt: res.RefreshExpiresAt,
	}
}

func mapMeResult(res *adminsvc.MeResult) *MeResult {
	if res == nil {
		return nil
	}
	return &MeResult{
		ID:              res.ID,
		Username:        res.Username,
		DisplayName:     res.DisplayName,
		Enabled:         res.Enabled,
		RoleCodes:       res.RoleCodes,
		PermissionCodes: res.PermissionCodes,
		CreatedAt:       res.CreatedAt,
		UpdatedAt:       res.UpdatedAt,
	}
}
