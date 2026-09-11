package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	platformauth "github.com/rio9466/easy-admin/server/internal/platform/auth"
	transporthttp "github.com/rio9466/easy-admin/server/internal/transport/http"
	"github.com/rio9466/easy-admin/server/internal/transport/http/handler"
	"github.com/rio9466/easy-admin/server/internal/transport/http/middleware"
	"github.com/rio9466/easy-admin/server/internal/transport/http/response"
)

type stubTokens struct {
	claims *platformauth.Claims
	err    error
}

func (s stubTokens) ParseAndValidate(string, platformauth.TokenType) (*platformauth.Claims, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.claims, nil
}

type stubSessions struct {
	sess *platformauth.Session
	err  error
}

func (s stubSessions) Get(context.Context, string) (*platformauth.Session, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.sess, nil
}

// stubAdminAuth implements handler.AdminAuthService for authz routing tests.
type stubAdminAuth struct {
	principal *middleware.AuthzPrincipal
	listCalls int
}

func (s *stubAdminAuth) LoadAuthz(context.Context, int64) (*middleware.AuthzPrincipal, error) {
	return s.principal, nil
}

func (s *stubAdminAuth) ListAdministrators(context.Context, handler.Actor, int, int, string) (adminauth.Page[adminauth.Administrator], error) {
	s.listCalls++
	return adminauth.Page[adminauth.Administrator]{Items: []adminauth.Administrator{}, Total: 0, Page: 1, PageSize: 20}, nil
}

func (s *stubAdminAuth) Login(context.Context, string, string, handler.Actor) (*handler.LoginResult, error) {
	return nil, apperr.ErrUnauthorized
}
func (s *stubAdminAuth) Refresh(context.Context, string, handler.Actor) (*handler.LoginResult, error) {
	return nil, apperr.ErrUnauthorized
}
func (s *stubAdminAuth) Logout(context.Context, handler.Actor) error { return nil }
func (s *stubAdminAuth) Me(context.Context, handler.Actor) (*handler.MeResult, error) {
	return nil, apperr.ErrUnauthorized
}
func (s *stubAdminAuth) ChangePassword(context.Context, handler.Actor, string, string) error {
	return apperr.ErrUnauthorized
}
func (s *stubAdminAuth) UpdateMyProfile(context.Context, handler.Actor, string) (*handler.MeResult, error) {
	return nil, apperr.ErrUnauthorized
}
func (s *stubAdminAuth) GetAdministrator(context.Context, handler.Actor, int64) (*adminauth.Administrator, error) {
	return nil, apperr.ErrNotFound
}
func (s *stubAdminAuth) CreateAdministrator(context.Context, handler.Actor, handler.CreateAdministratorInput) (*adminauth.Administrator, error) {
	return nil, apperr.ErrForbidden
}
func (s *stubAdminAuth) UpdateAdministrator(context.Context, handler.Actor, int64, handler.UpdateAdministratorInput) (*adminauth.Administrator, error) {
	return nil, apperr.ErrForbidden
}
func (s *stubAdminAuth) EnableAdministrator(context.Context, handler.Actor, int64) (*adminauth.Administrator, error) {
	return nil, apperr.ErrForbidden
}
func (s *stubAdminAuth) DisableAdministrator(context.Context, handler.Actor, int64) (*adminauth.Administrator, error) {
	return nil, apperr.ErrForbidden
}
func (s *stubAdminAuth) ResetAdministratorPassword(context.Context, handler.Actor, int64, string) error {
	return apperr.ErrForbidden
}
func (s *stubAdminAuth) AssignAdministratorRoles(context.Context, handler.Actor, int64, []string) (*adminauth.Administrator, error) {
	return nil, apperr.ErrForbidden
}
func (s *stubAdminAuth) ListRoles(context.Context, handler.Actor) ([]adminauth.Role, error) {
	return nil, apperr.ErrForbidden
}
func (s *stubAdminAuth) GetRole(context.Context, handler.Actor, int64) (*adminauth.Role, error) {
	return nil, apperr.ErrNotFound
}
func (s *stubAdminAuth) CreateRole(context.Context, handler.Actor, handler.CreateRoleInput) (*adminauth.Role, error) {
	return nil, apperr.ErrForbidden
}
func (s *stubAdminAuth) UpdateRole(context.Context, handler.Actor, int64, handler.UpdateRoleInput) (*adminauth.Role, error) {
	return nil, apperr.ErrForbidden
}
func (s *stubAdminAuth) ReplaceRolePermissions(context.Context, handler.Actor, int64, []string) (*adminauth.Role, error) {
	return nil, apperr.ErrForbidden
}
func (s *stubAdminAuth) ListPermissions(context.Context, handler.Actor) ([]adminauth.Permission, error) {
	return nil, apperr.ErrForbidden
}
func (s *stubAdminAuth) ListAuditEvents(context.Context, handler.Actor, handler.AuditListInput) (adminauth.Page[adminauth.AuditEvent], error) {
	return adminauth.Page[adminauth.AuditEvent]{}, apperr.ErrForbidden
}
func (s *stubAdminAuth) GetAuditEvent(context.Context, handler.Actor, int64) (*adminauth.AuditEvent, error) {
	return nil, apperr.ErrNotFound
}

func TestAdministratorsListRequiresAuthentication(t *testing.T) {
	t.Parallel()

	auth := &stubAdminAuth{}
	router := transporthttp.NewRouter(transporthttp.Dependencies{
		ReadyChecker: stubChecker{},
		AdminAuth:    auth,
		Tokens:       stubTokens{},
		Sessions:     stubSessions{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/administrators", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertAppError(t, rec, http.StatusUnauthorized, apperr.CodeUnauthorized)
	if auth.listCalls != 0 {
		t.Fatalf("list handler called %d times without auth", auth.listCalls)
	}
}

func TestAdministratorsListForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()

	auth := &stubAdminAuth{
		principal: &middleware.AuthzPrincipal{
			AdminID:         7,
			Username:        "finance",
			DisplayName:     "Finance",
			Enabled:         true,
			RoleCodes:       []string{adminauth.RoleFinance},
			PermissionCodes: []string{adminauth.PermDashboardView},
		},
	}
	router := transporthttp.NewRouter(transporthttp.Dependencies{
		ReadyChecker: stubChecker{},
		AdminAuth:    auth,
		Tokens: stubTokens{claims: &platformauth.Claims{
			RegisteredClaims: jwt.RegisteredClaims{Subject: "7"},
			SessionID:        "sid-7",
			TokenType:        platformauth.TokenTypeAccess,
		}},
		Sessions: stubSessions{sess: &platformauth.Session{AdminID: 7}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/administrators", nil)
	req.Header.Set("Authorization", "Bearer test-access")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertAppError(t, rec, http.StatusForbidden, apperr.CodeForbidden)
	if auth.listCalls != 0 {
		t.Fatalf("list handler called despite missing permission")
	}
}

func TestAdministratorsListAllowedWithPermission(t *testing.T) {
	t.Parallel()

	auth := &stubAdminAuth{
		principal: &middleware.AuthzPrincipal{
			AdminID:         1,
			Username:        "admin",
			Enabled:         true,
			RoleCodes:       []string{adminauth.RoleAdmin},
			PermissionCodes: []string{adminauth.PermAdminUserRead},
		},
	}
	router := transporthttp.NewRouter(transporthttp.Dependencies{
		ReadyChecker: stubChecker{},
		AdminAuth:    auth,
		Tokens: stubTokens{claims: &platformauth.Claims{
			RegisteredClaims: jwt.RegisteredClaims{Subject: "1"},
			SessionID:        "sid-1",
			TokenType:        platformauth.TokenTypeAccess,
		}},
		Sessions: stubSessions{sess: &platformauth.Session{AdminID: 1}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/administrators", nil)
	req.Header.Set("Authorization", "Bearer test-access")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if auth.listCalls != 1 {
		t.Fatalf("listCalls = %d, want 1", auth.listCalls)
	}
}

func assertAppError(t *testing.T, rec *httptest.ResponseRecorder, status, code int) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, status, rec.Body.String())
	}
	var env response.Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if env.Code != code {
		t.Fatalf("code = %d, want %d", env.Code, code)
	}
	if env.RequestID == "" {
		t.Fatal("request_id empty")
	}
}
