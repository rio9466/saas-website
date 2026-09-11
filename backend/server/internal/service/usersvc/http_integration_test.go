package usersvc_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	goredis "github.com/redis/go-redis/v9"
	"github.com/rio9466/easy-admin/server/internal/config"
	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	platformauth "github.com/rio9466/easy-admin/server/internal/platform/auth"
	"github.com/rio9466/easy-admin/server/internal/platform/mailer"
	platformpostgres "github.com/rio9466/easy-admin/server/internal/platform/postgres"
	"github.com/rio9466/easy-admin/server/internal/platform/ratelimit"
	"github.com/rio9466/easy-admin/server/internal/platform/secrets"
	"github.com/rio9466/easy-admin/server/internal/repository/logdb"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
	adminsvc "github.com/rio9466/easy-admin/server/internal/service/adminauth"
	usersvc "github.com/rio9466/easy-admin/server/internal/service/usersvc"
	transporthttp "github.com/rio9466/easy-admin/server/internal/transport/http"
	"github.com/rio9466/easy-admin/server/internal/transport/http/handler"
)

// realHTTPEnv wires the full router (admin + user surfaces) to isolated real
// stores and a fake mailer. One super administrator is bootstrapped when the
// DB has none (fresh package run); otherwise bootstrap refusal is tolerated.
type realHTTPEnv struct {
	t         *testing.T
	ctx       context.Context
	router    http.Handler
	users     *primary.UserRepository
	primaryDB *platformpostgres.DB
	logDB     *platformpostgres.DB
	mailer    *recordingMailer
	cfg       config.Config
}

func newRealHTTPEnv(t *testing.T) *realHTTPEnv {
	t.Helper()
	mustSkipIfUnavailable(t)
	cfg := mustLoadConfig(t)
	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)

	primaryDB, err := platformpostgres.Open(ctx, cfg.Database.Primary)
	if err != nil {
		t.Skipf("skip: primary postgres unavailable: %v", err)
	}
	logDB, err := platformpostgres.Open(ctx, cfg.Database.Log)
	if err != nil {
		_ = primaryDB.Close()
		t.Skipf("skip: log postgres unavailable: %v", err)
	}
	rdb := goredis.NewClient(&goredis.Options{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB})
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		_ = logDB.Close()
		_ = primaryDB.Close()
		t.Skipf("skip: redis unavailable: %v", err)
	}
	rec := &recordingMailer{}

	adminTokens, _ := platformauth.NewTokenService(platformauth.JWTConfig{Secret: cfg.Auth.JWTSecret, Issuer: cfg.Auth.JWTIssuer, Audience: cfg.Auth.JWTAudience})
	adminSessions, _ := platformauth.NewSessionStore(rdb)
	userTokens, _ := platformauth.NewTokenService(platformauth.JWTConfig{Secret: cfg.Auth.JWTSecret, Issuer: cfg.Auth.JWTIssuer, Audience: cfg.Auth.UserJWTAudience})
	userSessions, _ := platformauth.NewUserSessionStore(rdb)
	passwords, _ := platformauth.NewPasswordHasher(12)
	box, _ := secrets.NewBox(testMasterKey())
	limiter, _ := ratelimit.New(rdb)

	adminRepo := primary.NewAdminRepository(primaryDB.GORM())
	userRepo := primary.NewUserRepository(primaryDB.GORM())
	auditRepo := logdb.NewAuditRepository(logDB.GORM())

	adminSvc, err := adminsvc.New(adminRepo, auditRepo, adminTokens, adminSessions, passwords, logger, adminsvc.AuthOptions{Environment: "development"})
	if err != nil {
		t.Fatalf("admin svc: %v", err)
	}
	userSvc, err := usersvc.New(userRepo, adminRepo, auditRepo, userTokens, userSessions, passwords, box, limiter, logger, usersvc.Options{
		Environment: "development",
		NewMailer: func(mailer.Settings) (mailer.Mailer, error) {
			return rec, nil
		},
	})
	if err != nil {
		t.Fatalf("user svc: %v", err)
	}

	// Ensure one super admin exists (bootstrap is one-time; tolerate refusal
	// when a previous test in this package already seeded one).
	role, roleErr := adminRepo.GetRoleByCode(ctx, adminauth.RoleSuperAdmin)
	if roleErr == nil {
		_ = adminRepo.BootstrapSuperAdmin(ctx, &adminauth.Administrator{
			Username:     "http_root_admin",
			PasswordHash: "http-root-password-hash-not-used",
			DisplayName:  "HTTP Root",
			Enabled:      true,
		}, role.ID)
	}

	userAdapter := &handler.UserServiceAdapter{Svc: userSvc}
	router := transporthttp.NewRouter(transporthttp.Dependencies{
		Logger:         logger,
		AdminAuth:      &handler.ServiceAdapter{Svc: adminSvc},
		UserClient:     userAdapter,
		UserAdmin:      userAdapter,
		Tokens:         adminTokens,
		Sessions:       adminSessions,
		UserTokens:     userTokens,
		UserSessions:   userSessions,
		TrustedOrigins: []string{"http://localhost:3000", "http://localhost:8848", "http://127.0.0.1:8848"},
	})

	t.Cleanup(func() {
		_ = rdb.Close()
		_ = logDB.Close()
		_ = primaryDB.Close()
	})
	return &realHTTPEnv{t: t, ctx: ctx, router: router, users: userRepo, primaryDB: primaryDB, logDB: logDB, mailer: rec, cfg: cfg}
}

// envResponse summarizes an HTTP response for assertions.
type envResponse struct {
	status  int
	code    int
	body    map[string]any
	cookie  string
	headers http.Header
}

// nextHTTPAddr gives every HTTP request its own source address so fixed-window
// rate limits never couple independent tests (the server derives ClientIP from
// RemoteAddr, never from a client-supplied header).
func nextHTTPAddr() string {
	n := rateIPCounter.Add(1)
	return "10.80." + strconv.Itoa(int(n/256%200)) + "." + strconv.Itoa(int(n%256)) + ":12345"
}

func (e *realHTTPEnv) request(method, path, token, origin string, payload any) envResponse {
	e.t.Helper()
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			e.t.Fatal(err)
		}
		body = strings.NewReader(string(raw))
	}
	req := httptest.NewRequest(method, path, body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	req.RemoteAddr = nextHTTPAddr()
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	out := envResponse{status: rec.Code, headers: rec.Header()}
	for _, h := range rec.Result().Header.Values("Set-Cookie") {
		c, err := http.ParseSetCookie(h)
		if err != nil {
			continue
		}
		if c.Name == "ea_user_refresh" {
			out.cookie = c.Value
		}
	}
	if rec.Body.Len() > 0 {
		var envelope struct {
			Code int            `json:"code"`
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err == nil {
			out.code = envelope.Code
			out.body = envelope.Data
		}
	}
	return out
}

// setSettings updates one boolean column directly so HTTP tests start from a
// deterministic settings state regardless of prior tests in the package.
func (e *realHTTPEnv) setSettingsBool(t *testing.T, column string, value bool) {
	t.Helper()
	db := e.primaryDB.GORM()
	if err := db.Exec("UPDATE system_settings SET "+column+" = ?, version = version + 1 WHERE id = 1", value).Error; err != nil {
		t.Fatal(err)
	}
}

func (e *realHTTPEnv) registerUser(t *testing.T, username, origin string) {
	t.Helper()
	e.setSettingsBool(t, "registration_enabled", true)
	e.setSettingsBool(t, "username_login_enabled", true)
	e.setSettingsBool(t, "email_login_enabled", true)
	e.setSettingsBool(t, "email_verification_required", false)
	e.setSettingsBool(t, "smtp_enabled", false)
	res := e.request(http.MethodPost, "/api/v1/auth/register", "", origin, map[string]any{
		"username": username,
		"email":    username + "@example.com",
		"password": "password-123456",
	})
	if res.status != http.StatusCreated {
		t.Fatalf("register status=%d code=%d body=%v", res.status, res.code, res.body)
	}
}

func (e *realHTTPEnv) loginUser(t *testing.T, identifier, password, origin string) (access, refresh string) {
	t.Helper()
	res := e.request(http.MethodPost, "/api/v1/auth/login", "", origin, map[string]any{
		"identifier": identifier,
		"password":   password,
	})
	if res.status != http.StatusOK {
		t.Fatalf("login status=%d code=%d body=%v", res.status, res.code, res.body)
	}
	if res.cookie == "" {
		t.Fatal("login must set the user refresh cookie")
	}
	token, _ := res.body["access_token"].(string)
	if token == "" {
		t.Fatal("no access token returned")
	}
	return token, res.cookie
}

func TestUserRegistrationLoginMeRefreshLogoutHTTP(t *testing.T) {
	e := newRealHTTPEnv(t)
	origin := "http://localhost:3000"
	e.registerUser(t, "it_http_user", origin)

	access, refresh := e.loginUser(t, "it_http_user", "password-123456", origin)

	me := e.request(http.MethodGet, "/api/v1/me", access, "", nil)
	if me.status != http.StatusOK {
		t.Fatalf("me status=%d", me.status)
	}
	if pts, _ := me.body["points_balance"].(string); pts != "0.0000" {
		t.Fatalf("points_balance = %v, want exact 0.0000 string", me.body["points_balance"])
	}
	if _, ok := me.body["consumption_points"].(string); !ok {
		t.Fatal("consumption_points must be a fixed string")
	}

	// Refresh with the HttpOnly cookie.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "ea_user_refresh", Value: refresh})
	req.Header.Set("Origin", origin)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh status=%d body=%s", rec.Code, rec.Body.String())
	}

	// Logout revokes the session; the access token stops working.
	logout := e.request(http.MethodPost, "/api/v1/auth/logout", access, origin, nil)
	if logout.status != http.StatusOK {
		t.Fatalf("logout status=%d", logout.status)
	}
	after := e.request(http.MethodGet, "/api/v1/me", access, "", nil)
	if after.status != http.StatusUnauthorized {
		t.Fatalf("me after logout status=%d, want 401", after.status)
	}
}

func TestUserAndAdminTokensAreNotInterchangeable(t *testing.T) {
	e := newRealHTTPEnv(t)
	origin := "http://localhost:3000"
	e.registerUser(t, "it_sep_user", origin)
	userAccess, _ := e.loginUser(t, "it_sep_user", "password-123456", origin)

	// A user token must never reach an admin endpoint.
	adminList := e.request(http.MethodGet, "/api/v1/admin/users", userAccess, origin, nil)
	if adminList.status == http.StatusOK {
		t.Fatal("user token must not access admin endpoints")
	}

	// Anonymous /me → 401.
	anon := e.request(http.MethodGet, "/api/v1/me", "", "", nil)
	if anon.status != http.StatusUnauthorized {
		t.Fatalf("anonymous /me status=%d, want 401", anon.status)
	}
}

func TestRegistrationDisabledHTTP(t *testing.T) {
	e := newRealHTTPEnv(t)
	origin := "http://localhost:3000"
	e.setSettingsBool(t, "registration_enabled", false)
	res := e.request(http.MethodPost, "/api/v1/auth/register", "", origin, map[string]any{
		"username": "it_blocked_http", "email": "it_blocked_http@example.com", "password": "password-123456",
	})
	if res.status != http.StatusForbidden || res.code != apperr.CodeRegistrationDisabled {
		t.Fatalf("register disabled status=%d code=%d, want 403/40010", res.status, res.code)
	}
}

func TestPublicSettingsEndpointWhitelistHTTP(t *testing.T) {
	e := newRealHTTPEnv(t)
	res := e.request(http.MethodGet, "/api/v1/public/settings", "", "", nil)
	if res.status != http.StatusOK {
		t.Fatalf("public settings status=%d", res.status)
	}
	raw, _ := json.Marshal(res.body)
	lower := strings.ToLower(string(raw))
	for _, forbidden := range []string{"smtp", "password", "version"} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("public settings leaked %q: %s", forbidden, raw)
		}
	}
	for _, key := range []string{"platform_name", "registration_enabled", "username_login_enabled", "email_login_enabled", "email_verification_required"} {
		if _, ok := res.body[key]; !ok {
			t.Fatalf("public settings missing %q", key)
		}
	}
}

func TestLoginRateLimitedHTTP(t *testing.T) {
	e := newRealHTTPEnv(t)
	origin := "http://localhost:3000"
	// Exceed the fixed per-IP login window (httptest remote addr constant).
	for i := 0; i < 25; i++ {
		res := e.request(http.MethodPost, "/api/v1/auth/login", "", origin, map[string]any{
			"identifier": "it_missing_user", "password": "wrong-password",
		})
		if res.code == apperr.CodeRateLimited {
			return // fail-closed limit reached
		}
	}
	t.Fatal("login never hit the rate limit (fail-closed required)")
}

func TestLoginMethodDisabledHTTP(t *testing.T) {
	e := newRealHTTPEnv(t)
	origin := "http://localhost:3000"
	e.registerUser(t, "it_mode_user", origin)
	// Only email login stays enabled from here on.
	e.setSettingsBool(t, "username_login_enabled", false)
	e.setSettingsBool(t, "email_login_enabled", true)
	// Username login now rejected with the method-disabled error.
	res := e.request(http.MethodPost, "/api/v1/auth/login", "", origin, map[string]any{
		"identifier": "it_mode_user", "password": "password-123456",
	})
	if res.status != http.StatusBadRequest || res.code != apperr.CodeLoginMethodDisabled {
		t.Fatalf("disabled method status=%d code=%d, want 400/40012", res.status, res.code)
	}
}

// TestAdminBusinessUserEndpointsRequireAdminAuth proves the admin user/level/
// settings routes return 401 without an admin token (direct endpoint tests).
func TestAdminBusinessUserEndpointsRequireAdminAuth(t *testing.T) {
	e := newRealHTTPEnv(t)
	origin := "http://localhost:3000"
	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/admin/users"},
		{http.MethodPost, "/api/v1/admin/users"},
		{http.MethodGet, "/api/v1/admin/user-levels"},
		{http.MethodPost, "/api/v1/admin/user-levels"},
		{http.MethodGet, "/api/v1/admin/system-settings"},
		{http.MethodPut, "/api/v1/admin/system-settings"},
	} {
		res := e.request(tc.method, tc.path, "", origin, nil)
		if res.status != http.StatusUnauthorized && res.status != http.StatusForbidden {
			t.Fatalf("%s %s without admin auth status=%d, want 401", tc.method, tc.path, res.status)
		}
	}
}
