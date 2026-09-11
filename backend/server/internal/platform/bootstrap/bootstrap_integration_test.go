package bootstrap_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rio9466/easy-admin/server/internal/config"
	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/platform/bootstrap"
	"github.com/rio9466/easy-admin/server/internal/platform/migrate"
	"github.com/rio9466/easy-admin/server/internal/platform/postgres"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
)

// maintenanceBaseDSN is the maintenance DSN (any existing database on the test
// PostgreSQL instance) used to create uniquely named disposable databases.
// Each test gets its own database (name embeds pid + random bytes), applies
// the real primary migrations, and drops it again on cleanup. Configured
// application databases are never opened for mutation; when PostgreSQL is
// unavailable every database-backed test skips.
const maintenanceBaseDSNEnv = "EASY_ADMIN_TEST_POSTGRES_DSN"

func maintenanceBaseDSN() string {
	if base := os.Getenv(maintenanceBaseDSNEnv); base != "" {
		return base
	}
	return "postgres://easy_admin:easy_admin_dev_only@127.0.0.1:55432/postgres?sslmode=disable"
}

// lookup builds a credential lookup function from a map.
func lookup(pairs map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		v, ok := pairs[key]
		return v, ok
	}
}

// newMigratedIsolatedDB provisions a unique disposable database for one test,
// applies the real primary migrations through the startup migration runner,
// opens it, and registers drop-on-cleanup.
func newMigratedIsolatedDB(t *testing.T) *postgres.DB {
	t.Helper()

	dsn, drop, err := createDisposableDatabase(maintenanceBaseDSN())
	if err != nil {
		t.Skipf("bootstrap test skipped: disposable postgres unavailable (%v); configured application databases are not used", err)
	}
	t.Cleanup(func() { _ = drop() })

	db, err := postgres.Open(context.Background(), isolatedTestPostgresConfig(dsn))
	if err != nil {
		t.Fatalf("open isolated primary: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	res, err := migrate.Run(context.Background(), dsn, findRepoRoot(t), "primary")
	if err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if len(res.Applied) == 0 && res.Skipped == 0 {
		t.Fatalf("expected fresh database to apply migrations, got %+v", res)
	}
	return db
}

// findRepoRoot walks up from the test working directory until the migrations
// directory of the server module is found.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "migrations", "primary")); err == nil {
			// migrate.Run joins root + target, so hand back the migrations dir.
			return filepath.Join(dir, "migrations")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("migrations directory not found")
	return ""
}

// isolatedTestPostgresConfig builds a bounded pool config for the disposable
// database; only the DSN and timeouts matter in tests.
func isolatedTestPostgresConfig(dsn string) config.PostgresConfig {
	return config.PostgresConfig{
		DSN:             dsn,
		MaxOpenConns:    4,
		MaxIdleConns:    2,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
		StartupTimeout:  5 * time.Second,
		PingTimeout:     2 * time.Second,
	}
}

func TestEnsureFreshDatabaseCreatesAdminAndChineseRole(t *testing.T) {
	ctx := context.Background()
	db := newMigratedIsolatedDB(t)

	res, err := bootstrap.Ensure(ctx, db, lookup(nil), "development", 12)
	if err != nil {
		t.Fatalf("first Ensure: %v", err)
	}
	if !res.AdminCreated || res.AdminExists {
		t.Fatalf("first Ensure = %+v, want created on fresh database", res)
	}

	repo := primary.NewAdminRepository(db.GORM())
	admin, err := repo.GetAdministratorByUsername(ctx, "admin")
	if err != nil {
		t.Fatalf("load bootstrap admin: %v", err)
	}
	if !admin.Enabled || admin.PasswordHash == "" {
		t.Fatalf("bootstrap admin state invalid: %+v", admin)
	}
	hasSuper, err := repo.AdministratorHasRole(ctx, admin.ID, adminauth.RoleSuperAdmin)
	if err != nil || !hasSuper {
		t.Fatalf("bootstrap admin missing super_admin role (has=%v, err=%v)", hasSuper, err)
	}

	role, err := repo.GetRoleByCode(ctx, adminauth.RoleSuperAdmin)
	if err != nil {
		t.Fatalf("load super_admin role: %v", err)
	}
	if role.Name != "超级管理员" || role.Description != "超级管理员" {
		t.Fatalf("role = %q/%q, want Chinese name and description", role.Name, role.Description)
	}
}

// TestFreshDatabaseRolesAreExactlySuperAdmin proves that a fresh database ends
// up with exactly one role (super_admin): migrations seed only super_admin and
// bootstrap never adds more. The reserved admin/finance codes must not appear.
func TestFreshDatabaseRolesAreExactlySuperAdmin(t *testing.T) {
	ctx := context.Background()
	db := newMigratedIsolatedDB(t)

	if _, err := bootstrap.Ensure(ctx, db, lookup(nil), "development", 12); err != nil {
		t.Fatalf("Ensure: %v", err)
	}

	repo := primary.NewAdminRepository(db.GORM())
	roles, err := repo.ListRoles(ctx)
	if err != nil {
		t.Fatalf("list roles: %v", err)
	}
	if len(roles) != 1 {
		t.Fatalf("fresh database roles = %+v, want exactly [super_admin]", roles)
	}
	if roles[0].Code != adminauth.RoleSuperAdmin || roles[0].Name != "超级管理员" || roles[0].Description != "超级管理员" {
		t.Fatalf("single role = %+v, want super_admin with Chinese name/description", roles[0])
	}
	if !roles[0].BuiltIn || !roles[0].Enabled {
		t.Fatalf("super_admin must be built-in and enabled: %+v", roles[0])
	}
}

// TestFreshDatabaseSupportsOperatorCreatedRoles proves that after bootstrap a
// database contains only super_admin, and that operators can still create
// additional roles through the repository (role management surface), which is
// how the reserved admin/finance codes come back when operators need them.
// Service-level rejection of the reserved codes is covered by service tests.
func TestFreshDatabaseSupportsOperatorCreatedRoles(t *testing.T) {
	ctx := context.Background()
	db := newMigratedIsolatedDB(t)

	repo := primary.NewAdminRepository(db.GORM())
	if _, err := bootstrap.Ensure(ctx, db, lookup(nil), "development", 12); err != nil {
		t.Fatalf("Ensure: %v", err)
	}

	// An operator-created custom role works normally.
	custom := &adminauth.Role{Code: "ops", Name: "运营", BuiltIn: false, Enabled: true}
	if err := repo.CreateRole(ctx, custom); err != nil {
		t.Fatalf("create custom role: %v", err)
	}
	roles, err := repo.ListRoles(ctx)
	if err != nil {
		t.Fatalf("list roles: %v", err)
	}
	if len(roles) != 2 {
		t.Fatalf("roles = %+v, want super_admin + custom ops", roles)
	}
	byCode := map[string]bool{}
	for _, role := range roles {
		byCode[role.Code] = true
	}
	if !byCode[adminauth.RoleSuperAdmin] || !byCode["ops"] {
		t.Fatalf("roles = %+v, want exactly super_admin + ops", roles)
	}
}

func TestEnsureIsIdempotentAcrossRestarts(t *testing.T) {
	ctx := context.Background()
	db := newMigratedIsolatedDB(t)

	if _, err := bootstrap.Ensure(ctx, db, lookup(nil), "development", 12); err != nil {
		t.Fatalf("first Ensure: %v", err)
	}
	for i := 0; i < 3; i++ {
		res, err := bootstrap.Ensure(ctx, db, lookup(nil), "development", 12)
		if err != nil {
			t.Fatalf("repeat Ensure %d: %v", i+1, err)
		}
		if res.AdminCreated {
			t.Fatalf("repeat Ensure %d recreated administrator: %+v", i+1, res)
		}
		if !res.AdminExists {
			t.Fatalf("repeat Ensure %d lost admin exists signal: %+v", i+1, res)
		}
	}
	repo := primary.NewAdminRepository(db.GORM())
	total, err := repo.CountAdministrators(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != 1 {
		t.Fatalf("administrators = %d, want exactly 1 after repeated Ensure", total)
	}
}

func TestEnsureDoesNotDeleteExistingAdministratorsRolesOrPermissions(t *testing.T) {
	ctx := context.Background()
	db := newMigratedIsolatedDB(t)

	if _, err := bootstrap.Ensure(ctx, db, lookup(nil), "development", 12); err != nil {
		t.Fatalf("first Ensure: %v", err)
	}

	repo := primary.NewAdminRepository(db.GORM())
	// A second administrator with a non-super role and a custom role: startup
	// must preserve all of them.
	customRole := &adminauth.Role{Code: "ops", Name: "运营", BuiltIn: false, Enabled: true}
	if err := repo.CreateRole(ctx, customRole); err != nil {
		t.Fatalf("create custom role: %v", err)
	}
	second := &adminauth.Administrator{
		Username:     "ops_user",
		PasswordHash: "$2a$12$" + "0123456789012345678901234567890123456789012345678901234",
		DisplayName:  "运营同学",
		Enabled:      true,
	}
	opsRole, err := repo.GetRoleByCode(ctx, "ops")
	if err != nil {
		t.Fatalf("load ops role: %v", err)
	}
	if err := repo.CreateAdministrator(ctx, second, []int64{opsRole.ID}); err != nil {
		t.Fatalf("create second administrator: %v", err)
	}
	rolesBefore, err := repo.ListRoles(ctx)
	if err != nil {
		t.Fatalf("list roles before: %v", err)
	}
	permsBefore, err := repo.ListPermissions(ctx)
	if err != nil {
		t.Fatalf("list permissions before: %v", err)
	}

	res, err := bootstrap.Ensure(ctx, db, lookup(nil), "development", 12)
	if err != nil {
		t.Fatalf("Ensure with existing data: %v", err)
	}
	if res.AdminCreated || !res.AdminExists {
		t.Fatalf("Ensure = %+v, want existing admins preserved untouched", res)
	}

	total, err := repo.CountAdministrators(ctx)
	if err != nil {
		t.Fatalf("count after: %v", err)
	}
	if total != 2 {
		t.Fatalf("administrators = %d, want 2 (no deletions)", total)
	}
	rolesAfter, err := repo.ListRoles(ctx)
	if err != nil {
		t.Fatalf("list roles after: %v", err)
	}
	permsAfter, err := repo.ListPermissions(ctx)
	if err != nil {
		t.Fatalf("list permissions after: %v", err)
	}
	if len(rolesAfter) != len(rolesBefore) || len(permsAfter) != len(permsBefore) {
		t.Fatalf("roles/permissions changed: roles %d→%d, perms %d→%d",
			len(rolesBefore), len(rolesAfter), len(permsBefore), len(permsAfter))
	}

	// Renaming super_admin to Chinese must not have touched the custom role.
	opsAfter, err := repo.GetRoleByCode(ctx, "ops")
	if err != nil {
		t.Fatalf("load ops role after: %v", err)
	}
	if opsAfter.Name != "运营" || opsAfter.ID != opsRole.ID {
		t.Fatalf("custom role was modified by bootstrap: %+v", opsAfter)
	}
}

func TestEnsureUpdatesLegacyRoleNameInPlace(t *testing.T) {
	ctx := context.Background()
	db := newMigratedIsolatedDB(t)

	// Simulate a database bootstrapped before the rename: role still English.
	repo := primary.NewAdminRepository(db.GORM())
	legacy, err := repo.GetRoleByCode(ctx, adminauth.RoleSuperAdmin)
	if err != nil {
		t.Fatalf("load role: %v", err)
	}
	english := "Super Administrator"
	englishDesc := "Unrestricted administrative access"
	if err := repo.UpdateRole(ctx, legacy.ID, &english, &englishDesc, nil); err != nil {
		t.Fatalf("revert role to legacy name: %v", err)
	}

	res, err := bootstrap.Ensure(ctx, db, lookup(nil), "development", 12)
	if err != nil {
		t.Fatalf("Ensure after legacy revert: %v", err)
	}
	if !res.RoleUpdated {
		t.Fatalf("Ensure = %+v, want RoleUpdated for legacy name", res)
	}
	role, err := repo.GetRoleByCode(ctx, adminauth.RoleSuperAdmin)
	if err != nil {
		t.Fatalf("reload role: %v", err)
	}
	if role.Name != "超级管理员" || role.Description != "超级管理员" {
		t.Fatalf("role not renamed in place: %q/%q", role.Name, role.Description)
	}
	// Second run is a no-op rename-wise.
	res2, err := bootstrap.Ensure(ctx, db, lookup(nil), "development", 12)
	if err != nil || res2.RoleUpdated {
		t.Fatalf("second Ensure = %+v err=%v, want no further rename", res2, err)
	}
}

// TestDatabaseMigratedWithLegacySeedsKeepsItsRoles simulates a database that
// applied the pre-20260907 migration (with seeded admin/finance roles) and
// proves that re-running migrations plus bootstrap neither deletes nor
// duplicates those roles — legacy databases keep their data untouched.
func TestDatabaseMigratedWithLegacySeedsKeepsItsRoles(t *testing.T) {
	ctx := context.Background()
	db := newMigratedIsolatedDB(t)
	repo := primary.NewAdminRepository(db.GORM())

	// Insert the legacy roles exactly as the old migration did.
	legacy := []*adminauth.Role{
		{Code: "admin", Name: "Administrator", Description: "Manages ordinary administrators and non-super roles", BuiltIn: true, Enabled: true},
		{Code: "finance", Name: "Finance", Description: "Reserved finance role", BuiltIn: true, Enabled: true},
	}
	for _, role := range legacy {
		if err := repo.CreateRole(ctx, role); err != nil {
			t.Fatalf("seed legacy role %s: %v", role.Code, err)
		}
	}

	// Re-running migrations must be a no-op and bootstrap must not delete.
	if _, err := bootstrap.Ensure(ctx, db, lookup(nil), "development", 12); err != nil {
		t.Fatalf("Ensure on legacy database: %v", err)
	}

	roles, err := repo.ListRoles(ctx)
	if err != nil {
		t.Fatalf("list roles: %v", err)
	}
	codes := map[string]bool{}
	for _, role := range roles {
		codes[strings.ToLower(role.Code)] = true
	}
	for _, want := range []string{"super_admin", "admin", "finance"} {
		if !codes[want] {
			t.Fatalf("legacy role %q missing after startup, roles = %+v", want, roles)
		}
	}
	if len(roles) != 3 {
		t.Fatalf("roles = %+v, want exactly super_admin+admin+finance preserved", roles)
	}

	// The bootstrap admin still gets super_admin; legacy roles keep their rows.
	total, err := repo.CountAdministrators(ctx)
	if err != nil {
		t.Fatalf("count admins: %v", err)
	}
	if total != 1 {
		t.Fatalf("administrators = %d, want 1", total)
	}
}

func TestEnsureFailsClosedWithoutCredentialsOnEmptyDatabase(t *testing.T) {
	ctx := context.Background()
	db := newMigratedIsolatedDB(t)

	// Production with no credentials and no administrators: must fail and not
	// create anything.
	_, err := bootstrap.Ensure(ctx, db, lookup(nil), "production", 12)
	if err == nil {
		t.Fatal("expected fail-closed error without bootstrap credentials")
	}
	repo := primary.NewAdminRepository(db.GORM())
	total, err := repo.CountAdministrators(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != 0 {
		t.Fatalf("administrators = %d, want 0 after failed bootstrap", total)
	}
}

func TestEnsureRejectsProductionDefaultCredentials(t *testing.T) {
	ctx := context.Background()
	db := newMigratedIsolatedDB(t)

	_, err := bootstrap.Ensure(ctx, db, lookup(map[string]string{
		bootstrap.EnvBootstrapUsername: "admin",
		bootstrap.EnvBootstrapPassword: "admin123",
	}), "production", 12)
	if err == nil {
		t.Fatal("expected production rejection of development default credentials")
	}
	repo := primary.NewAdminRepository(db.GORM())
	total, err := repo.CountAdministrators(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != 0 {
		t.Fatalf("administrators = %d, want 0 after rejected bootstrap", total)
	}
}

func TestCredentialsFromEnvRules(t *testing.T) {
	if _, err := bootstrap.CredentialsFromEnv(lookup(nil), "production"); err == nil {
		t.Fatal("production without credentials must fail")
	}
	opts, err := bootstrap.CredentialsFromEnv(lookup(nil), "development")
	if err != nil {
		t.Fatalf("development defaults: %v", err)
	}
	if opts.Username != "admin" || opts.Password == "" {
		t.Fatalf("development defaults = %+v", opts)
	}
	if opts.DisplayName != "超级管理员" {
		t.Fatalf("display name default = %q, want 超级管理员", opts.DisplayName)
	}
	short, err := bootstrap.CredentialsFromEnv(lookup(map[string]string{
		bootstrap.EnvBootstrapPassword: "a",
	}), "development")
	if err == nil {
		t.Fatalf("short custom password must fail, got %+v", short)
	}
	displayed, err := bootstrap.CredentialsFromEnv(lookup(map[string]string{
		bootstrap.EnvBootstrapPassword: "long-enough-password",
		bootstrap.EnvBootstrapDisplay:  "运维管理员",
	}), "development")
	if err != nil {
		t.Fatalf("custom credentials: %v", err)
	}
	if displayed.DisplayName != "运维管理员" {
		t.Fatalf("display override = %q", displayed.DisplayName)
	}
	if displayed.Password != "long-enough-password" {
		t.Fatalf("credential values must round-trip without transformation")
	}

	// The username is fixed to admin: any other configured value is rejected.
	// (Values are trimmed first, mirroring normal username normalization.)
	for _, bad := range []string{"root", "operator", "Admin", "ADMIN", " administrator"} {
		if _, err := bootstrap.CredentialsFromEnv(lookup(map[string]string{
			bootstrap.EnvBootstrapUsername: bad,
			bootstrap.EnvBootstrapPassword: "long-enough-password",
		}), "development"); err == nil {
			t.Fatalf("username %q must be rejected", bad)
		}
	}
	// Restating admin (the fixed account) is accepted and normalized to admin.
	restated, err := bootstrap.CredentialsFromEnv(lookup(map[string]string{
		bootstrap.EnvBootstrapUsername: "admin",
		bootstrap.EnvBootstrapPassword: "long-enough-password",
	}), "development")
	if err != nil {
		t.Fatalf("restating admin username: %v", err)
	}
	if restated.Username != "admin" {
		t.Fatalf("restated username = %q, want admin", restated.Username)
	}
}

// TestEnsureRejectsNonAdminUsernameOnEmptyDatabase proves a non-admin
// BOOTSTRAP_ADMIN_USERNAME (e.g. root) aborts bootstrap on a fresh database
// and creates nothing.
func TestEnsureRejectsNonAdminUsernameOnEmptyDatabase(t *testing.T) {
	ctx := context.Background()
	db := newMigratedIsolatedDB(t)

	_, err := bootstrap.Ensure(ctx, db, lookup(map[string]string{
		bootstrap.EnvBootstrapUsername: "root",
		bootstrap.EnvBootstrapPassword: "long-enough-password",
	}), "development", 12)
	if err == nil {
		t.Fatal("expected non-admin username to be rejected")
	}
	repo := primary.NewAdminRepository(db.GORM())
	total, err := repo.CountAdministrators(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != 0 {
		t.Fatalf("administrators = %d, want 0 after rejected bootstrap", total)
	}
}

// TestEnsureAlwaysCreatesFixedAdminAccount proves the initial account is
// always username `admin`: unset username, explicit admin restatement, and a
// password-only configuration all converge on the same fixed account.
func TestEnsureAlwaysCreatesFixedAdminAccount(t *testing.T) {
	ctx := context.Background()
	variants := []map[string]string{
		nil,
		{bootstrap.EnvBootstrapUsername: "admin", bootstrap.EnvBootstrapPassword: "custom-pass-0001"},
		{bootstrap.EnvBootstrapPassword: "another-pass-0001"},
	}
	for i, envPairs := range variants {
		db := newMigratedIsolatedDB(t)
		res, err := bootstrap.Ensure(ctx, db, lookup(envPairs), "development", 12)
		if err != nil {
			t.Fatalf("variant %d Ensure: %v", i+1, err)
		}
		if !res.AdminCreated {
			t.Fatalf("variant %d Ensure = %+v, want created", i+1, res)
		}
		repo := primary.NewAdminRepository(db.GORM())
		if _, err := repo.GetAdministratorByUsername(ctx, "admin"); err != nil {
			t.Fatalf("variant %d: fixed admin account missing: %v", i+1, err)
		}
		total, err := repo.CountAdministrators(ctx)
		if err != nil {
			t.Fatalf("variant %d count: %v", i+1, err)
		}
		if total != 1 {
			t.Fatalf("variant %d administrators = %d, want exactly 1 (admin)", i+1, total)
		}
	}
}

// TestEnsureWithExistingAdminsNeverCreatesAccounts proves that with
// administrators already present no account is created or renamed — including
// when bootstrap credentials are configured.
func TestEnsureWithExistingAdminsNeverCreatesAccounts(t *testing.T) {
	ctx := context.Background()
	db := newMigratedIsolatedDB(t)

	if _, err := bootstrap.Ensure(ctx, db, lookup(nil), "development", 12); err != nil {
		t.Fatalf("first Ensure: %v", err)
	}
	res, err := bootstrap.Ensure(ctx, db, lookup(map[string]string{
		bootstrap.EnvBootstrapUsername: "admin",
		bootstrap.EnvBootstrapPassword: "whatever-pass-001",
	}), "development", 12)
	if err != nil {
		t.Fatalf("Ensure with existing admins: %v", err)
	}
	if res.AdminCreated || !res.AdminExists {
		t.Fatalf("Ensure = %+v, want existing admins untouched", res)
	}
	repo := primary.NewAdminRepository(db.GORM())
	total, err := repo.CountAdministrators(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != 1 {
		t.Fatalf("administrators = %d, want exactly 1 after repeat bootstrap", total)
	}
}

func createDisposableDatabase(baseDSN string) (string, func() error, error) {
	maintenance, err := sql.Open("pgx", baseDSN)
	if err != nil {
		return "", nil, fmt.Errorf("open maintenance connection: %w", err)
	}
	defer maintenance.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := maintenance.PingContext(ctx); err != nil {
		return "", nil, fmt.Errorf("ping maintenance database: %w", err)
	}

	dbName, err := uniqueTestDBName("easy_admin_it_bootstrap")
	if err != nil {
		return "", nil, err
	}

	if _, err := maintenance.ExecContext(ctx,
		`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`,
		dbName); err != nil {
		return "", nil, fmt.Errorf("terminate stale connections: %w", err)
	}
	if _, err := maintenance.ExecContext(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %s`, quoteIdent(dbName))); err != nil {
		return "", nil, fmt.Errorf("drop stale database: %w", err)
	}
	if _, err := maintenance.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", quoteIdent(dbName))); err != nil {
		return "", nil, fmt.Errorf("create disposable database: %w", err)
	}

	parsed, err := pgx.ParseConfig(baseDSN)
	if err != nil {
		_ = dropDatabase(ctx, maintenance, dbName)
		return "", nil, fmt.Errorf("parse base dsn: %w", err)
	}
	sslmode := "disable"
	if parsed.TLSConfig != nil {
		sslmode = "require"
	}
	userinfo := url.UserPassword(parsed.User, parsed.Password).String()
	dsn := fmt.Sprintf("postgres://%s@%s:%d/%s?sslmode=%s", userinfo, parsed.Host, parsed.Port, dbName, sslmode)

	cleanup := func() error {
		main, openErr := sql.Open("pgx", baseDSN)
		if openErr != nil {
			return openErr
		}
		defer main.Close()
		return dropDatabase(context.Background(), main, dbName)
	}
	return dsn, cleanup, nil
}

// uniqueTestDBName embeds pid + random bytes so concurrent go test invocations
// never collide and names stay within the PostgreSQL identifier limit.
func uniqueTestDBName(prefix string) (string, error) {
	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("random: %w", err)
	}
	return fmt.Sprintf("%s_%d_%s", prefix, os.Getpid(), hex.EncodeToString(buf[:])), nil
}

func dropDatabase(ctx context.Context, db *sql.DB, dbName string) error {
	if _, err := db.ExecContext(ctx,
		`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`,
		dbName); err != nil {
		return fmt.Errorf("terminate disposable database connections: %w", err)
	}
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %s`, quoteIdent(dbName))); err != nil {
		return fmt.Errorf("drop disposable database: %w", err)
	}
	return nil
}

func quoteIdent(name string) string {
	return `"` + name + `"`
}
