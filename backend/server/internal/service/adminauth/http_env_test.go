package adminauth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	platformauth "github.com/rio9466/easy-admin/server/internal/platform/auth"
	platformpostgres "github.com/rio9466/easy-admin/server/internal/platform/postgres"
	"github.com/rio9466/easy-admin/server/internal/repository/logdb"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
	svc "github.com/rio9466/easy-admin/server/internal/service/adminauth"
	transporthttp "github.com/rio9466/easy-admin/server/internal/transport/http"
	"github.com/rio9466/easy-admin/server/internal/transport/http/handler"
	"golang.org/x/crypto/bcrypt"
)

type readyStub struct{}

func (readyStub) Ping(context.Context) error { return nil }

// httpEnv wires the real HTTP router to real primary PostgreSQL, log
// PostgreSQL, and Redis so session-invalidation and role-disable behavior is
// proven against production semantics rather than substitutes.
type httpEnv struct {
	t         *testing.T
	ctx       context.Context
	router    http.Handler
	admins    *primary.AdminRepository
	audits    *logdb.AuditRepository
	sessions  *platformauth.SessionStore
	tokens    *platformauth.TokenService
	primaryDB *platformpostgres.DB
	logDB     *platformpostgres.DB
	redis     *goredis.Client
}

func newHTTPEnv(t *testing.T) *httpEnv {
	t.Helper()
	cfg := mustLoadConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	primaryDB, err := platformpostgres.Open(ctx, cfg.Database.Primary)
	if err != nil {
		t.Skipf("skip real-store test: primary postgres unavailable: %v", err)
	}
	logDB, err := platformpostgres.Open(ctx, cfg.Database.Log)
	if err != nil {
		_ = primaryDB.Close()
		t.Skipf("skip real-store test: log postgres unavailable: %v", err)
	}
	rdb := goredis.NewClient(&goredis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		_ = logDB.Close()
		_ = primaryDB.Close()
		t.Skipf("skip real-store test: redis unavailable: %v", err)
	}

	sessions, err := platformauth.NewSessionStore(rdb)
	if err != nil {
		t.Fatalf("session store: %v", err)
	}
	tokens, err := platformauth.NewTokenService(platformauth.JWTConfig{
		Secret:   cfg.Auth.JWTSecret,
		Issuer:   cfg.Auth.JWTIssuer,
		Audience: cfg.Auth.JWTAudience,
	})
	if err != nil {
		t.Fatalf("tokens: %v", err)
	}
	passwords, err := platformauth.NewPasswordHasher(cfg.Auth.BcryptCost)
	if err != nil {
		t.Fatalf("passwords: %v", err)
	}

	adminRepo := primary.NewAdminRepository(primaryDB.GORM())
	auditRepo := logdb.NewAuditRepository(logDB.GORM())
	service, err := svc.New(adminRepo, auditRepo, tokens, sessions, passwords, slog.Default(), svc.AuthOptions{
		Environment: "development",
	})
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}
	adapter := &handler.ServiceAdapter{Svc: service}
	router := transporthttp.NewRouter(transporthttp.Dependencies{
		ReadyChecker: readyStub{},
		AdminAuth:    adapter,
		Tokens:       tokens,
		Sessions:     sessions,
		Cookie: handler.AuthCookieSettings{
			Name:   "ea_admin_refresh",
			Secure: false,
		},
	})

	t.Cleanup(func() {
		_ = rdb.Close()
		_ = logDB.Close()
		_ = primaryDB.Close()
	})
	return &httpEnv{
		t:         t,
		ctx:       context.Background(),
		router:    router,
		admins:    adminRepo,
		audits:    auditRepo,
		sessions:  sessions,
		tokens:    tokens,
		primaryDB: primaryDB,
		logDB:     logDB,
		redis:     rdb,
	}
}

func (e *httpEnv) resetRBAC(t *testing.T) {
	t.Helper()
	db := e.primaryDB.GORM()
	sqls := []string{
		`DELETE FROM role_permissions WHERE role_id IN (SELECT id FROM roles WHERE code LIKE 'it_%')`,
		`DELETE FROM administrator_roles WHERE administrator_id IN (SELECT id FROM administrators WHERE username LIKE 'it_%')`,
		`DELETE FROM roles WHERE code LIKE 'it_%'`,
		`TRUNCATE administrator_roles, administrators RESTART IDENTITY CASCADE`,
	}
	for _, q := range sqls {
		if err := db.Exec(q).Error; err != nil {
			t.Fatalf("resetRBAC: %v", err)
		}
	}
}

// createAdmin inserts an enabled administrator with the given role codes.
// Reserved roles (admin/finance) are created on demand because startup
// migrations no longer seed them on fresh databases.
func (e *httpEnv) createAdmin(t *testing.T, username, password string, roleCodes []string) int64 {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	roleIDs := make([]int64, 0, len(roleCodes))
	for _, code := range roleCodes {
		role := seedReservedRole(t, e.admins, e.ctx, code)
		roleIDs = append(roleIDs, role.ID)
	}
	admin := &adminauth.Administrator{
		Username:     username,
		PasswordHash: string(hash),
		DisplayName:  username,
		Enabled:      true,
	}
	if err := e.admins.CreateAdministrator(e.ctx, admin, roleIDs); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	return admin.ID
}

type envResponse struct {
	status int
	code   int
	body   map[string]any
	cookie string
	maxAge int
}

func (e *httpEnv) do(t *testing.T, method, path, token string, payload any) envResponse {
	t.Helper()
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		body = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return decodeEnvResponse(t, rec)
}

func decodeEnvResponse(t *testing.T, rec *httptest.ResponseRecorder) envResponse {
	t.Helper()
	out := envResponse{status: rec.Code}
	for _, header := range rec.Result().Header.Values("Set-Cookie") {
		cookie, err := http.ParseSetCookie(header)
		if err != nil {
			continue
		}
		if cookie.Name == "ea_admin_refresh" {
			out.cookie = cookie.Value
			out.maxAge = cookie.MaxAge
		}
	}
	if rec.Body.Len() == 0 {
		return out
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out.body); err != nil {
		t.Fatalf("decode envelope: %v (body=%s)", err, rec.Body.String())
	}
	if code, ok := out.body["code"].(float64); ok {
		out.code = int(code)
	}
	return out
}

func (e *httpEnv) login(t *testing.T, username, password string) (token, refresh string) {
	t.Helper()
	res := e.do(t, http.MethodPost, "/api/v1/admin/auth/login", "", map[string]any{
		"username": username,
		"password": password,
	})
	if res.status != http.StatusOK {
		t.Fatalf("login %s status=%d body=%v", username, res.status, res.body)
	}
	data, _ := res.body["data"].(map[string]any)
	token, _ = data["access_token"].(string)
	if token == "" {
		t.Fatalf("login %s returned no access token", username)
	}
	return token, res.cookie
}

func (e *httpEnv) loginExpect(t *testing.T, username, password string, wantStatus int) envResponse {
	t.Helper()
	return e.do(t, http.MethodPost, "/api/v1/admin/auth/login", "", map[string]any{
		"username": username,
		"password": password,
	})
}

func (e *httpEnv) refresh(t *testing.T, refreshToken string) envResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/refresh", nil)
	if refreshToken != "" {
		req.AddCookie(&http.Cookie{Name: "ea_admin_refresh", Value: refreshToken})
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return decodeEnvResponse(t, rec)
}

func (e *httpEnv) me(t *testing.T, token string) envResponse {
	t.Helper()
	return e.do(t, http.MethodGet, "/api/v1/admin/me", token, nil)
}

// mintSessionForEpoch creates an active Redis session for adminID with an
// explicit auth epoch and returns an access/refresh pair bound to it. This
// simulates a login that a Redis revocation scan could miss because it was
// issued concurrently with an epoch bump.
func (e *httpEnv) mintSessionForEpoch(t *testing.T, adminID, epoch int64) (access, refresh, sid string) {
	t.Helper()
	sid = "it-synth-" + strings.ReplaceAll(time.Now().UTC().Format("150405.000000000"), ".", "") + "-" + t.Name()
	expiresAt := time.Now().UTC().Add(platformauth.RefreshTokenTTL)
	if err := e.sessions.Create(e.ctx, sid, adminID, platformauth.HashRefreshVerifier("synth-jti"), epoch, expiresAt); err != nil {
		t.Fatalf("mint session: %v", err)
	}
	subject := idString(adminID)
	access, _, err := e.tokens.IssueAccess(subject, sid)
	if err != nil {
		t.Fatalf("mint access: %v", err)
	}
	refresh, _, err = e.tokens.IssueRefresh(subject, sid)
	if err != nil {
		t.Fatalf("mint refresh: %v", err)
	}
	return access, refresh, sid
}

func idString(id int64) string {
	return strconv.FormatInt(id, 10)
}

func dataString(envelope map[string]any, key string) string {
	data, _ := envelope["data"].(map[string]any)
	s, _ := data[key].(string)
	return s
}

func timeNowNanos() int64 {
	return time.Now().UnixNano()
}

func requireNotSuccess(t *testing.T, name string, res envResponse) {
	t.Helper()
	if res.status == http.StatusOK || res.code == 0 {
		t.Fatalf("%s unexpectedly succeeded: status=%d body=%v", name, res.status, res.body)
	}
}
