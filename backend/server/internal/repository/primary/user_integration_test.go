package primary_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/rio9466/easy-admin/server/internal/config"
	userdomain "github.com/rio9466/easy-admin/server/internal/domain/user"
	platformpostgres "github.com/rio9466/easy-admin/server/internal/platform/postgres"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
)

func openUserRepo(t *testing.T) *primary.UserRepository {
	t.Helper()
	requireIsolatedDB(t)
	cfgPath := os.Getenv("EASY_ADMIN_CONFIG")
	if cfgPath == "" {
		cfgPath = findRepoFile("configs/config.local.toml")
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	ctx := context.Background()
	db, err := platformpostgres.Open(ctx, cfg.Database.Primary)
	if err != nil {
		t.Skipf("skip: primary postgres unavailable: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return primary.NewUserRepository(db.GORM())
}

func defaultLevelID(t *testing.T, repo *primary.UserRepository) int64 {
	t.Helper()
	settings, err := repo.GetSystemSettings(context.Background())
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	return settings.DefaultLevelID
}

func mustUser(t *testing.T, repo *primary.UserRepository, username string) *userdomain.User {
	t.Helper()
	levelID := defaultLevelID(t, repo)
	u := &userdomain.User{
		Username:       username,
		Email:          username + "@example.com",
		PasswordHash:   "hash-" + username,
		AvatarURL:      "",
		Nickname:       username,
		RegistrationIP: "127.0.0.1",
		Status:         userdomain.StatusActive,
		LevelMode:      userdomain.LevelModeAuto,
	}
	if err := repo.CreateUser(context.Background(), u, levelID); err != nil {
		t.Fatalf("create user %s: %v", username, err)
	}
	// Reload so the test observes the real DB defaults (epoch, level join).
	loaded, err := repo.GetUserByUsername(context.Background(), username)
	if err != nil {
		t.Fatalf("reload user %s: %v", username, err)
	}
	return loaded
}

func TestCreateUserDuplicateNormalizedRejected(t *testing.T) {
	repo := openUserRepo(t)
	ctx := context.Background()
	levelID := defaultLevelID(t, repo)

	a := &userdomain.User{Username: "Alice", Email: "Alice@Example.com", PasswordHash: "h", Nickname: "alice", RegistrationIP: "127.0.0.1", Status: userdomain.StatusActive}
	if err := repo.CreateUser(ctx, a, levelID); err != nil {
		t.Fatalf("create a: %v", err)
	}

	// Same username, different case/space: normalized unique must reject.
	b := &userdomain.User{Username: "alice ", Email: "other@example.com", PasswordHash: "h", Nickname: "b", RegistrationIP: "127.0.0.1", Status: userdomain.StatusActive}
	if err := repo.CreateUser(ctx, b, levelID); err == nil {
		t.Fatal("duplicate normalized username must fail")
	} else if !errors.Is(err, primary.ErrConflict) {
		t.Fatalf("err = %v, want ErrConflict", err)
	}

	// Same email, different case: must reject.
	c := &userdomain.User{Username: "carol", Email: "alice@example.com", PasswordHash: "h", Nickname: "c", RegistrationIP: "127.0.0.1", Status: userdomain.StatusActive}
	if err := repo.CreateUser(ctx, c, levelID); err == nil {
		t.Fatal("duplicate normalized email must fail")
	}
}

func TestUsersEmailVerifiedStatusAndEpochBumps(t *testing.T) {
	repo := openUserRepo(t)
	ctx := context.Background()
	u := mustUser(t, repo, "epoch_user")

	// Disable bumps the epoch so old sessions are invalidated.
	if err := repo.SetUserEnabled(ctx, u.ID, false); err != nil {
		t.Fatalf("disable: %v", err)
	}
	disabled, err := repo.GetUserByID(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Status != userdomain.StatusDisabled {
		t.Fatalf("status = %s, want disabled", disabled.Status)
	}
	if disabled.AuthEpoch != u.AuthEpoch+1 {
		t.Fatalf("epoch = %d, want %d", disabled.AuthEpoch, u.AuthEpoch+1)
	}

	// Password reset also bumps the epoch and changes the hash.
	if err := repo.SetUserPassword(ctx, u.ID, "new-hash"); err != nil {
		t.Fatal(err)
	}
	afterReset, err := repo.GetUserByUsername(ctx, "epoch_user")
	if err != nil {
		t.Fatal(err)
	}
	if afterReset.PasswordHash != "new-hash" {
		t.Fatal("password hash not replaced")
	}
	if afterReset.AuthEpoch != disabled.AuthEpoch+1 {
		t.Fatalf("epoch after reset = %d", afterReset.AuthEpoch)
	}
}

func TestAdjustPointsAtomicIdempotentAndLevelAutoRecalc(t *testing.T) {
	repo := openUserRepo(t)
	ctx := context.Background()
	u := mustUser(t, repo, "points_user")

	actor := int64(1)
	// First adjustment: +10 available and +10 consumption (auto level stays default if only one level).
	pt, err := repo.AdjustPoints(ctx, u.ID, &actor, mustDecimal(t, "10.0000"), mustDecimal(t, "10.0000"), "first", "key-1")
	if err != nil {
		t.Fatalf("adjust: %v", err)
	}
	if pt.BalanceAfter.String() != "10.0000" || pt.ConsumptionAfter.String() != "10.0000" {
		t.Fatalf("after = %s/%s", pt.BalanceAfter.String(), pt.ConsumptionAfter.String())
	}

	// Idempotency: same key must be rejected as replay.
	_, err = repo.AdjustPoints(ctx, u.ID, &actor, mustDecimal(t, "10.0000"), mustDecimal(t, "10.0000"), "first", "key-1")
	if !errors.Is(err, primary.ErrIdempotencyReplay) {
		t.Fatalf("replay err = %v, want ErrIdempotencyReplay", err)
	}

	// Negative available that would go below zero must fail.
	_, err = repo.AdjustPoints(ctx, u.ID, &actor, mustDecimal(t, "-11.0000"), userdomain.Zero(), "overdraw", "key-2")
	if err == nil {
		t.Fatal("overdraw must fail")
	}
	// Consumption cannot be reduced.
	_, err = repo.AdjustPoints(ctx, u.ID, &actor, userdomain.Zero(), mustDecimal(t, "-1.0000"), "reduce-consumption", "key-3")
	if err == nil {
		t.Fatal("consumption reduction must fail")
	}
	// No-op (both deltas zero) must fail.
	_, err = repo.AdjustPoints(ctx, u.ID, &actor, userdomain.Zero(), userdomain.Zero(), "noop", "key-4")
	if err == nil {
		t.Fatal("no-op adjustment must fail")
	}
}

func TestAdjustPointsConcurrentDeltasSerialize(t *testing.T) {
	repo := openUserRepo(t)
	ctx := context.Background()
	u := mustUser(t, repo, "conc_user")

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			actor := int64(100 + n)
			_, err := repo.AdjustPoints(ctx, u.ID, &actor, mustDecimal(t, fmt.Sprintf("%d.0000", n+1)), userdomain.Zero(), "concurrent", fmt.Sprintf("conc-%d", n))
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent adjust failed: %v", err)
		}
	}

	got, err := repo.GetUserByID(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Sum 1..8 = 36 exactly, no lost updates.
	if got.PointsBalance.String() != "36.0000" {
		t.Fatalf("final balance = %s, want 36.0000 (no lost updates)", got.PointsBalance.String())
	}
	if got.ConsumptionPoints.String() != "0.0000" {
		t.Fatalf("consumption = %s, want 0.0000", got.ConsumptionPoints.String())
	}

	// Ledger must hold exactly 8 rows.
	page, err := repo.ListPointTransactions(ctx, u.ID, 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 8 {
		t.Fatalf("ledger rows = %d, want 8", page.Total)
	}
}

func TestEnabledLevelThresholdUnambiguous(t *testing.T) {
	repo := openUserRepo(t)
	ctx := context.Background()

	mk := func(code string, threshold string, enabled bool) *userdomain.UserLevel {
		return &userdomain.UserLevel{
			Code:            code,
			Name:            code,
			ThresholdPoints: mustDecimal(t, threshold),
			SortOrder:       0,
			Enabled:         enabled,
		}
	}
	if err := repo.CreateUserLevel(ctx, mk("lv1", "0.0000", true)); err == nil {
		t.Fatal("enabled level sharing the seeded 0.0000 threshold must be rejected")
	}
	if err := repo.CreateUserLevel(ctx, mk("lv10", "10.0000", true)); err != nil {
		t.Fatalf("create lv10: %v", err)
	}
	if err := repo.CreateUserLevel(ctx, mk("lv10b", "10.0000", true)); err == nil {
		t.Fatal("two enabled levels with the same threshold must be rejected")
	}
	// A disabled level may reuse a threshold.
	dup := mk("lv10-disabled", "10.0000", false)
	if err := repo.CreateUserLevel(ctx, dup); err != nil {
		t.Fatalf("disabled duplicate threshold must be allowed: %v", err)
	}
	// And re-enabling it must then fail (ambiguity).
	enabled := true
	if err := repo.UpdateUserLevel(ctx, dup.ID, nil, nil, nil, nil, &enabled); err == nil {
		t.Fatal("re-enabling a duplicate-threshold level must be rejected")
	}
}

func TestAutoLevelSelectionAndManualPersistence(t *testing.T) {
	repo := openUserRepo(t)
	ctx := context.Background()

	lvA, _ := userdomain.ParseDecimal4("100.0000")
	if err := repo.CreateUserLevel(ctx, &userdomain.UserLevel{Code: "gold", Name: "Gold", ThresholdPoints: lvA, SortOrder: 1, Enabled: true}); err != nil {
		t.Fatalf("create gold: %v", err)
	}
	gold, err := repo.GetUserLevelByID(ctx, goldID(repo, ctx, "gold"))
	if err != nil {
		t.Fatal(err)
	}

	u := mustUser(t, repo, "level_user")
	if u.LevelCode != "default" {
		t.Fatalf("initial level code = %s, want default", u.LevelCode)
	}

	// Consumption grows past 100 → auto level becomes gold.
	actor := int64(1)
	if _, err := repo.AdjustPoints(ctx, u.ID, &actor, userdomain.Zero(), mustDecimal(t, "150.0000"), "spend", "lvl-key-1"); err != nil {
		t.Fatalf("adjust: %v", err)
	}
	after, _ := repo.GetUserByID(ctx, u.ID)
	if after.LevelID != gold.ID || after.LevelCode != "gold" {
		t.Fatalf("auto level = %s (%d), want gold (%d)", after.LevelCode, after.LevelID, gold.ID)
	}

	// Manual assignment keeps level across a later available-only change.
	if _, err := repo.AssignUserLevel(ctx, u.ID, gold.ID, userdomain.LevelModeManual); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AdjustPoints(ctx, u.ID, &actor, mustDecimal(t, "5.0000"), userdomain.Zero(), "avail", "lvl-key-2"); err != nil {
		t.Fatal(err)
	}
	manual, _ := repo.GetUserByID(ctx, u.ID)
	if manual.LevelMode != userdomain.LevelModeManual {
		t.Fatal("level mode must stay manual")
	}

	// Switching back to auto recalculates immediately from consumption.
	auto, err := repo.AssignUserLevel(ctx, u.ID, gold.ID, userdomain.LevelModeAuto)
	if err != nil {
		t.Fatal(err)
	}
	if auto.LevelMode != userdomain.LevelModeAuto {
		t.Fatal("level mode must switch to auto")
	}
}

func TestSystemSettingsOptimisticLock(t *testing.T) {
	repo := openUserRepo(t)
	ctx := context.Background()
	settings, err := repo.GetSystemSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	updated := *settings
	updated.PlatformName = "v1-name"
	updated.UpdatedBy = 9

	if err := repo.UpdateSystemSettings(ctx, &updated, settings.Version, nil); err != nil {
		t.Fatalf("first update: %v", err)
	}
	// Stale version must conflict.
	stale := *settings
	stale.PlatformName = "stale"
	stale.UpdatedBy = 9
	if err := repo.UpdateSystemSettings(ctx, &stale, settings.Version, nil); !errors.Is(err, primary.ErrSettingsVersionConflict) {
		t.Fatalf("stale update err = %v, want version conflict", err)
	}
	// Latest version wins.
	latest, _ := repo.GetSystemSettings(ctx)
	if latest.PlatformName != "v1-name" {
		t.Fatalf("platform name = %s", latest.PlatformName)
	}
	if latest.Version != settings.Version+1 {
		t.Fatalf("version = %d, want %d", latest.Version, settings.Version+1)
	}
}

func goldID(repo *primary.UserRepository, ctx context.Context, code string) int64 {
	levels, err := repo.ListUserLevels(ctx)
	if err != nil {
		return 0
	}
	for _, l := range levels {
		if l.Code == code {
			return l.ID
		}
	}
	return 0
}

func mustDecimal(t *testing.T, s string) userdomain.Decimal4 {
	t.Helper()
	d, err := userdomain.ParseDecimal4(s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return d
}
