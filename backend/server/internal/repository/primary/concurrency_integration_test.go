package primary_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rio9466/easy-admin/server/internal/config"
	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/platform/postgres"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestBootstrapSuperAdminConcurrentOnlyOneWins(t *testing.T) {
	db, repo, cleanup := openPrimaryRepo(t)
	defer cleanup()
	ctx := context.Background()

	truncateAdministrators(t, db)

	role, err := repo.GetRoleByCode(ctx, adminauth.RoleSuperAdmin)
	if err != nil {
		t.Fatalf("GetRoleByCode: %v", err)
	}

	const workers = 8
	var success atomic.Int64
	var conflicts atomic.Int64
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		i := i
		go func() {
			defer wg.Done()
			hash, err := bcrypt.GenerateFromPassword([]byte(fmt.Sprintf("bootstrap-pass-%02d!!", i)), 12)
			if err != nil {
				t.Errorf("hash: %v", err)
				return
			}
			admin := &adminauth.Administrator{
				Username:     fmt.Sprintf("boot_concurrent_%d_%d", time.Now().UnixNano(), i),
				PasswordHash: string(hash),
				DisplayName:  "Boot",
				Enabled:      true,
			}
			err = repo.BootstrapSuperAdmin(ctx, admin, role.ID)
			switch {
			case err == nil:
				success.Add(1)
			case errors.Is(err, primary.ErrBootstrapAlreadyCompleted):
				conflicts.Add(1)
			default:
				t.Errorf("unexpected bootstrap error: %v", err)
			}
		}()
	}
	wg.Wait()

	if success.Load() != 1 {
		t.Fatalf("successes = %d, want 1", success.Load())
	}
	if conflicts.Load() != int64(workers-1) {
		t.Fatalf("conflicts = %d, want %d", conflicts.Load(), workers-1)
	}
	count, err := repo.CountAdministrators(ctx)
	if err != nil {
		t.Fatalf("CountAdministrators: %v", err)
	}
	if count != 1 {
		t.Fatalf("administrator count = %d, want 1", count)
	}
}

func TestDisableLastSuperAdminConcurrentSafe(t *testing.T) {
	db, repo, cleanup := openPrimaryRepo(t)
	defer cleanup()
	ctx := context.Background()
	truncateAdministrators(t, db)

	role, err := repo.GetRoleByCode(ctx, adminauth.RoleSuperAdmin)
	if err != nil {
		t.Fatalf("GetRoleByCode: %v", err)
	}
	adminRole := seedReservedRole(t, repo, ctx, adminauth.RoleAdmin)

	supers := make([]*adminauth.Administrator, 2)
	for i := 0; i < 2; i++ {
		hash, err := bcrypt.GenerateFromPassword([]byte(fmt.Sprintf("super-pass-%d-xxxx", i)), 12)
		if err != nil {
			t.Fatalf("hash: %v", err)
		}
		admin := &adminauth.Administrator{
			Username:     fmt.Sprintf("super_guard_%d_%d", time.Now().UnixNano(), i),
			PasswordHash: string(hash),
			DisplayName:  "Super",
			Enabled:      true,
		}
		if err := repo.CreateAdministrator(ctx, admin, []int64{role.ID}); err != nil {
			t.Fatalf("CreateAdministrator: %v", err)
		}
		supers[i] = admin
	}

	// First disable one super successfully so exactly one remains.
	if err := repo.DisableAdministratorGuardingLastSuper(ctx, supers[0].ID); err != nil {
		t.Fatalf("disable first super: %v", err)
	}

	var success atomic.Int64
	var blocked atomic.Int64
	var wg sync.WaitGroup
	const workers = 8
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			err := repo.DisableAdministratorGuardingLastSuper(ctx, supers[1].ID)
			switch {
			case err == nil:
				success.Add(1)
			case errors.Is(err, primary.ErrLastSuperAdmin):
				blocked.Add(1)
			default:
				t.Errorf("unexpected disable error: %v", err)
			}
		}()
	}
	wg.Wait()

	if success.Load() != 0 {
		t.Fatalf("last super disable successes = %d, want 0", success.Load())
	}
	if blocked.Load() != int64(workers) {
		t.Fatalf("blocked = %d, want %d", blocked.Load(), workers)
	}
	count, err := repo.CountEnabledSuperAdmins(ctx)
	if err != nil {
		t.Fatalf("CountEnabledSuperAdmins: %v", err)
	}
	if count != 1 {
		t.Fatalf("enabled supers = %d, want 1", count)
	}

	// Concurrent role removal with two supers should leave at least one.
	truncateAdministrators(t, db)
	for i := 0; i < 2; i++ {
		hash, err := bcrypt.GenerateFromPassword([]byte(fmt.Sprintf("super-role-%d-xxxx", i)), 12)
		if err != nil {
			t.Fatalf("hash: %v", err)
		}
		admin := &adminauth.Administrator{
			Username:     fmt.Sprintf("super_role_%d_%d", time.Now().UnixNano(), i),
			PasswordHash: string(hash),
			DisplayName:  "Super",
			Enabled:      true,
		}
		if err := repo.CreateAdministrator(ctx, admin, []int64{role.ID}); err != nil {
			t.Fatalf("CreateAdministrator: %v", err)
		}
		supers[i] = admin
	}

	success.Store(0)
	blocked.Store(0)
	wg.Add(2)
	for _, target := range supers {
		target := target
		go func() {
			defer wg.Done()
			err := repo.ReplaceAdministratorRolesGuardingLastSuper(ctx, target.ID, []int64{adminRole.ID})
			switch {
			case err == nil:
				success.Add(1)
			case errors.Is(err, primary.ErrLastSuperAdmin):
				blocked.Add(1)
			default:
				t.Errorf("unexpected replace roles error: %v", err)
			}
		}()
	}
	wg.Wait()

	if success.Load()+blocked.Load() != 2 {
		t.Fatalf("role-removal outcomes incomplete: success=%d blocked=%d", success.Load(), blocked.Load())
	}
	if success.Load() > 1 {
		t.Fatalf("role-removal successes = %d, want at most 1", success.Load())
	}
	count, err = repo.CountEnabledSuperAdmins(ctx)
	if err != nil {
		t.Fatalf("CountEnabledSuperAdmins after role race: %v", err)
	}
	if count < 1 {
		t.Fatalf("enabled supers after role race = %d, want >= 1", count)
	}
}

func openPrimaryRepo(t *testing.T) (*gorm.DB, *primary.AdminRepository, func()) {
	t.Helper()
	requireIsolatedDB(t)
	cfgPath := os.Getenv("EASY_ADMIN_CONFIG")
	if cfgPath == "" {
		cfgPath = filepathFromServerRoot(t, "configs/config.local.toml")
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Skipf("skip integration test: load config: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := postgres.Open(ctx, cfg.Database.Primary)
	if err != nil {
		t.Skipf("skip integration test: primary postgres unavailable: %v", err)
	}
	return db.GORM(), primary.NewAdminRepository(db.GORM()), func() { _ = db.Close() }
}

func truncateAdministrators(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`TRUNCATE administrator_roles, administrators RESTART IDENTITY CASCADE`).Error; err != nil {
		t.Fatalf("truncate administrators: %v", err)
	}
}

func filepathFromServerRoot(t *testing.T, rel string) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, rel)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Join(wd, "..", "..", rel)
}
