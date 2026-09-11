package primary_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/rio9466/easy-admin/server/internal/config"
	"github.com/rio9466/easy-admin/server/internal/domain/contact"
	"github.com/rio9466/easy-admin/server/internal/domain/media"
	platformpostgres "github.com/rio9466/easy-admin/server/internal/platform/postgres"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
)

func openInboxRepo(t *testing.T) *primary.InboxRepository {
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
	return primary.NewInboxRepository(db.GORM())
}

func TestInboxContactSubmissionLifecycle(t *testing.T) {
	repo := openInboxRepo(t)
	ctx := context.Background()

	sub := &contact.Submission{
		Name:      "Alice",
		Email:     "alice@example.com",
		Company:   "ACME",
		Message:   "Hello",
		Locale:    "en",
		Status:    contact.StatusNew,
		SourceIP:  "203.0.113.5",
		UserAgent: "test-agent",
	}
	if err := repo.CreateContactSubmission(ctx, sub); err != nil {
		t.Fatalf("create submission: %v", err)
	}
	if sub.ID == 0 {
		t.Fatal("submission id not assigned")
	}

	newItems, newTotal, err := repo.ListContactSubmissions(ctx, contact.StatusNew, 1, 10)
	if err != nil {
		t.Fatalf("list new: %v", err)
	}
	if newTotal != 1 || len(newItems) != 1 || newItems[0].ID != sub.ID {
		t.Fatalf("list new = %d items (total %d), want exactly the created submission", len(newItems), newTotal)
	}

	if err := repo.UpdateContactSubmissionStatus(ctx, sub.ID, contact.StatusRead); err != nil {
		t.Fatalf("update status: %v", err)
	}
	reloaded, err := repo.GetContactSubmission(ctx, sub.ID)
	if err != nil {
		t.Fatalf("get submission: %v", err)
	}
	if reloaded.Status != contact.StatusRead {
		t.Fatalf("status = %q, want read", reloaded.Status)
	}
	if reloaded.Name != sub.Name || reloaded.Email != sub.Email || reloaded.Message != sub.Message {
		t.Fatalf("status-only update mutated other fields: %#v", reloaded)
	}

	readItems, readTotal, err := repo.ListContactSubmissions(ctx, contact.StatusRead, 1, 10)
	if err != nil {
		t.Fatalf("list read: %v", err)
	}
	if readTotal != 1 || len(readItems) != 1 {
		t.Fatalf("list read total = %d, want 1", readTotal)
	}
	newAfter, newAfterTotal, err := repo.ListContactSubmissions(ctx, contact.StatusNew, 1, 10)
	if err != nil {
		t.Fatalf("list new after update: %v", err)
	}
	if newAfterTotal != 0 || len(newAfter) != 0 {
		t.Fatalf("list new after update total = %d, want 0", newAfterTotal)
	}

	// Pagination: create two more and page one-at-a-time, newest first.
	for i := 0; i < 2; i++ {
		extra := &contact.Submission{Name: "Bob", Email: "bob@example.com", Message: "hi", Status: contact.StatusNew}
		if err := repo.CreateContactSubmission(ctx, extra); err != nil {
			t.Fatalf("create extra: %v", err)
		}
	}
	page1, total, err := repo.ListContactSubmissions(ctx, "", 1, 2)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if total != 3 || len(page1) != 2 {
		t.Fatalf("page 1 = %d items (total %d), want 2 of 3", len(page1), total)
	}
	page2, _, err := repo.ListContactSubmissions(ctx, "", 2, 2)
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}
	if len(page2) != 1 {
		t.Fatalf("page 2 = %d items, want 1", len(page2))
	}
	if page1[0].ID == page2[0].ID {
		t.Fatal("pagination returned overlapping rows")
	}

	if _, err := repo.GetContactSubmission(ctx, sub.ID+100000); !errors.Is(err, primary.ErrNotFound) {
		t.Fatalf("get missing = %v, want ErrNotFound", err)
	}
}

func TestInboxMediaAssetLifecycle(t *testing.T) {
	repo := openInboxRepo(t)
	ctx := context.Background()

	asset := &media.Asset{
		URL:          "/media/" + uniqueTestDBName("asset") + ".png",
		OriginalName: "logo.png",
		MIME:         media.MIMEPNG,
		SizeBytes:    128,
		Width:        3,
		Height:       2,
		CreatedBy:    1,
	}
	if err := repo.CreateMediaAsset(ctx, asset); err != nil {
		t.Fatalf("create asset: %v", err)
	}
	loaded, err := repo.GetMediaAsset(ctx, asset.ID)
	if err != nil {
		t.Fatalf("get asset: %v", err)
	}
	if loaded.MIME != media.MIMEPNG || loaded.Width != 3 || loaded.Height != 2 || loaded.SizeBytes != 128 {
		t.Fatalf("asset round trip: %#v", loaded)
	}

	items, total, err := repo.ListMediaAssets(ctx, 1, 10)
	if err != nil {
		t.Fatalf("list assets: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != asset.ID {
		t.Fatalf("list assets = %d items (total %d), want exactly the created asset", len(items), total)
	}

	if err := repo.DeleteMediaAsset(ctx, asset.ID); err != nil {
		t.Fatalf("delete asset: %v", err)
	}
	if _, err := repo.GetMediaAsset(ctx, asset.ID); !errors.Is(err, primary.ErrNotFound) {
		t.Fatalf("get deleted = %v, want ErrNotFound", err)
	}
}
