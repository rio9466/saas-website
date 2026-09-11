package usersvc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/rio9466/easy-admin/server/internal/config"
	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	userdomain "github.com/rio9466/easy-admin/server/internal/domain/user"
	platformauth "github.com/rio9466/easy-admin/server/internal/platform/auth"
	"github.com/rio9466/easy-admin/server/internal/platform/mailer"
	platformpostgres "github.com/rio9466/easy-admin/server/internal/platform/postgres"
	"github.com/rio9466/easy-admin/server/internal/platform/ratelimit"
	"github.com/rio9466/easy-admin/server/internal/platform/secrets"
	"github.com/rio9466/easy-admin/server/internal/repository/logdb"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
	usersvc "github.com/rio9466/easy-admin/server/internal/service/usersvc"
)

// rateIPCounter hands every test request a unique source IP so fixed-window
// rate limits never couple independent tests.
var rateIPCounter atomic.Int64

func nextIP() string {
	n := rateIPCounter.Add(1)
	switch {
	case n < 256:
		return "10.90.0." + strconv.Itoa(int(n))
	case n < 65536:
		return "10.90." + strconv.Itoa(int((n-256)/256)) + "." + strconv.Itoa(int((n-256)%256))
	default:
		return "10." + strconv.Itoa(int((n/65536)%200+1)) + ".0." + strconv.Itoa(int(n%65536))
	}
}

func testMasterKey() []byte { return []byte("0123456789abcdef0123456789abcdef") }

// recordingMailer is an in-process mailer.Mailer that never sends external
// mail. It records recipients and bodies so tests can assert verification
// links were produced and that no credentials appear in the message.
type recordingMailer struct {
	mu       sync.Mutex
	sent     []mailRecord
	failNext error
}

type mailRecord struct {
	To      string
	Subject string
	Body    string
}

func (m *recordingMailer) Send(_ context.Context, to, subject, htmlBody string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failNext != nil {
		err := m.failNext
		m.failNext = nil
		return err
	}
	m.sent = append(m.sent, mailRecord{To: to, Subject: subject, Body: htmlBody})
	return nil
}

func (m *recordingMailer) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sent)
}

// reset clears recorded messages so a test can assert on its own sends only.
func (m *recordingMailer) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = nil
}

func (m *recordingMailer) last() mailRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sent) == 0 {
		return mailRecord{}
	}
	return m.sent[len(m.sent)-1]
}

// svcEnv wires the real usersvc Service to isolated primary/log PostgreSQL and
// a dedicated Redis database.
type svcEnv struct {
	t          *testing.T
	ctx        context.Context
	users      *primary.UserRepository
	admins     *primary.AdminRepository
	audits     *logdb.AuditRepository
	sessions   *platformauth.SessionStore
	tokens     *platformauth.TokenService
	svc        *usersvc.Service
	mailer     *recordingMailer
	box        *secrets.Box
	primaryDB  *platformpostgres.DB
	logDB      *platformpostgres.DB
	rdb        *goredis.Client
	actorAdmin usersvc.Actor
}

func mustLoadConfig(t *testing.T) config.Config {
	t.Helper()
	cfgPath := ""
	if p := env("EASY_ADMIN_CONFIG"); p != "" {
		cfgPath = p
	}
	if cfgPath == "" {
		cfgPath = findRepoFile("configs/config.local.toml")
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	return cfg
}

func env(key string) string {
	return os.Getenv(key)
}

func newSvcEnv(t *testing.T) *svcEnv {
	t.Helper()
	mustSkipIfUnavailable(t)
	cfg := mustLoadConfig(t)

	ctx := context.Background()
	primaryDB, err := platformpostgres.Open(ctx, cfg.Database.Primary)
	if err != nil {
		t.Skipf("skip real-store test: primary postgres unavailable: %v", err)
	}
	logDB, err := platformpostgres.Open(ctx, cfg.Database.Log)
	if err != nil {
		_ = primaryDB.Close()
		t.Skipf("skip real-store test: log postgres unavailable: %v", err)
	}
	rdb := goredis.NewClient(&goredis.Options{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB})
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		_ = logDB.Close()
		_ = primaryDB.Close()
		t.Skipf("skip real-store test: redis unavailable: %v", err)
	}

	tokens, err := platformauth.NewTokenService(platformauth.JWTConfig{
		Secret:   cfg.Auth.JWTSecret,
		Issuer:   cfg.Auth.JWTIssuer,
		Audience: "easy-admin-user",
	})
	if err != nil {
		t.Fatalf("user tokens: %v", err)
	}
	sessions, err := platformauth.NewUserSessionStore(rdb)
	if err != nil {
		t.Fatalf("user session store: %v", err)
	}
	box, err := secrets.NewBox(testMasterKey())
	if err != nil {
		t.Fatalf("box: %v", err)
	}
	limiter, err := ratelimit.New(rdb)
	if err != nil {
		t.Fatalf("limiter: %v", err)
	}
	passwords, err := platformauth.NewPasswordHasher(12)
	if err != nil {
		t.Fatalf("passwords: %v", err)
	}

	users := primary.NewUserRepository(primaryDB.GORM())
	admins := primary.NewAdminRepository(primaryDB.GORM())
	audits := logdb.NewAuditRepository(logDB.GORM())
	rec := &recordingMailer{}
	svc, err := usersvc.New(users, admins, audits, tokens, sessions, passwords, box, limiter, slog.New(slog.DiscardHandler), usersvc.Options{
		Environment: "development",
		NewMailer: func(mailer.Settings) (mailer.Mailer, error) {
			return rec, nil
		},
	})
	if err != nil {
		t.Fatalf("new user service: %v", err)
	}

	env := &svcEnv{
		t:         t,
		ctx:       ctx,
		users:     users,
		admins:    admins,
		audits:    audits,
		sessions:  sessions,
		tokens:    tokens,
		svc:       svc,
		mailer:    rec,
		box:       box,
		primaryDB: primaryDB,
		logDB:     logDB,
		rdb:       rdb,
	}

	// Seed a real super_admin so RBAC reads from primary succeed, then reset
	// settings to the migrated baseline so every test starts deterministic.
	env.seedSuperAdmin(t)
	env.resetSettingsBaseline(t)

	t.Cleanup(func() {
		_ = rdb.Close()
		_ = logDB.Close()
		_ = primaryDB.Close()
	})
	return env
}

// seedSuperAdmin creates an enabled super-admin administrator in the isolated
// primary DB so admin permission checks (read live from RBAC tables) pass.
func (e *svcEnv) seedSuperAdmin(t *testing.T) {
	t.Helper()
	role, err := e.admins.GetRoleByCode(e.ctx, adminauth.RoleSuperAdmin)
	if err != nil {
		t.Fatalf("super_admin role: %v", err)
	}
	username := "it_root_" + sanitizeTestName(t.Name())
	admin := &adminauth.Administrator{
		Username:     username,
		PasswordHash: "unused-bcrypt-hash-not-used",
		DisplayName:  "Test Root",
		Enabled:      true,
	}
	if err := e.admins.CreateAdministrator(e.ctx, admin, []int64{role.ID}); err != nil {
		t.Fatalf("create super admin: %v", err)
	}
	e.actorAdmin = usersvc.Actor{
		AdminID:          admin.ID,
		AdminUsername:    admin.Username,
		AdminDisplayName: admin.DisplayName,
		AdminRoleCodes:   []string{adminauth.RoleSuperAdmin},
		SourceIP:         nextIP(),
		UserAgent:        "test",
	}
}

// resetSettingsBaseline restores the migrated defaults: registration and both
// login modes enabled, verification disabled, SMTP disabled, points zero.
func (e *svcEnv) resetSettingsBaseline(t *testing.T) {
	t.Helper()
	cur, err := e.users.GetSystemSettings(e.ctx)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	_, err = e.svc.UpdateSystemSettings(e.ctx, e.actorAdmin, usersvc.UpdateSystemSettingsInput{
		PlatformName:              cur.PlatformName,
		PublicFrontendURL:         cur.PublicFrontendURL,
		PublicAPIURL:              cur.PublicAPIURL,
		RegistrationEnabled:       true,
		UsernameLoginEnabled:      true,
		EmailLoginEnabled:         true,
		EmailVerificationRequired: false,
		DefaultLevelID:            cur.DefaultLevelID,
		DefaultAvatarURL:          cur.DefaultAvatarURL,
		RegistrationPoints:        cur.RegistrationPoints,
		SMTPEnabled:               false,
		SMTPTLSMode:               cur.SMTPTLSMode,
		Version:                   cur.Version,
	})
	if err != nil {
		t.Fatalf("reset settings baseline: %v", err)
	}
}

func sanitizeTestName(name string) string {
	replacer := strings.NewReplacer("/", "_", " ", "_", "(", "", ")", "")
	s := replacer.Replace(name)
	if len(s) > 30 {
		s = s[len(s)-30:]
	}
	return s
}

// cleanUserData removes business-user rows created by a test so each test
// starts from the migrated baseline (settings singleton and default level stay).
func (e *svcEnv) cleanUserData(t *testing.T) {
	t.Helper()
	db := e.primaryDB.GORM()
	for _, q := range []string{
		`DELETE FROM user_point_transactions WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'it\_%')`,
		`DELETE FROM users WHERE username LIKE 'it\_%'`,
		`DELETE FROM user_levels WHERE code LIKE 'it\_lv\_%'`,
	} {
		if err := db.Exec(q).Error; err != nil {
			t.Fatalf("cleanUserData: %v", err)
		}
	}
}

// seedEnabledSMTP turns on SMTP + email verification through the service so
// verification flows run against the fake mailer.
func (e *svcEnv) seedEnabledSMTP(t *testing.T) {
	t.Helper()
	cur, err := e.svc.GetSystemSettingsForActor(e.ctx, e.actorAdmin)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	pass := "smtp-secret-password"
	_, err = e.svc.UpdateSystemSettings(e.ctx, e.actorAdmin, usersvc.UpdateSystemSettingsInput{
		PlatformName:              cur.PlatformName,
		PublicFrontendURL:         "http://localhost:3000",
		PublicAPIURL:              cur.PublicAPIURL,
		RegistrationEnabled:       true,
		UsernameLoginEnabled:      true,
		EmailLoginEnabled:         true,
		EmailVerificationRequired: true,
		DefaultLevelID:            cur.DefaultLevelID,
		DefaultAvatarURL:          cur.DefaultAvatarURL,
		RegistrationPoints:        mustDecimal(t, "0.0000"),
		SMTPEnabled:               true,
		SMTPHost:                  "smtp.fake.local",
		SMTPPort:                  587,
		SMTPUsername:              "smtp-user",
		SMTPPassword:              &pass,
		SMTPFromEmail:             "no-reply@easy-admin.local",
		SMTPFromName:              "easy-admin",
		SMTPTLSMode:               "starttls",
		Version:                   cur.Version,
	})
	if err != nil {
		t.Fatalf("enable smtp: %v", err)
	}
}

func (e *svcEnv) defaultLevelID(t *testing.T) int64 {
	t.Helper()
	s, err := e.users.GetSystemSettings(e.ctx)
	if err != nil {
		t.Fatal(err)
	}
	return s.DefaultLevelID
}

func (e *svcEnv) createActiveUser(t *testing.T, username, password string) *userdomain.User {
	t.Helper()
	u, err := e.svc.Register(e.ctx, e.registerActor(), usersvc.RegisterInput{
		Username: username,
		Email:    username + "@example.com",
		Password: password,
		SourceIP: nextIP(),
	})
	if err != nil {
		t.Fatalf("register %s: %v", username, err)
	}
	return u
}

func (e *svcEnv) registerActor() usersvc.Actor {
	return usersvc.Actor{SourceIP: nextIP(), UserAgent: "test"}
}

func (e *svcEnv) anonymousActor() usersvc.Actor {
	return usersvc.Actor{SourceIP: nextIP(), UserAgent: "test"}
}

func errCode(err error) int {
	if ae, ok := err.(*apperr.AppError); ok {
		return ae.Code
	}
	return 0
}

func bodyJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func mustDecimal(t *testing.T, s string) userdomain.Decimal4 {
	t.Helper()
	d, err := userdomain.ParseDecimal4(s)
	if err != nil {
		t.Fatalf("parse decimal %q: %v", s, err)
	}
	return d
}

func postJSON(router http.Handler, path string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func containsStr(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func idFromString(t *testing.T, s string) int64 {
	t.Helper()
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		t.Fatalf("parse id %q: %v", s, err)
	}
	return n
}

func waitFor(t *testing.T, cond func() bool, what string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", what)
}

// errMailerForcedFailure is a stable sentinel used by tests to make exactly one
// mailer send fail without touching any external SMTP service.
var errMailerForcedFailure = errors.New("forced mailer failure")
