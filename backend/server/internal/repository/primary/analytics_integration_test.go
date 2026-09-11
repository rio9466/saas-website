package primary_test

import (
	"context"
	"testing"
	"time"

	"github.com/rio9466/easy-admin/server/internal/domain/analytics"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
)

// TestPageViewRepositoryInsertAndOverview covers ingest plus the window
// aggregation (pv, distinct source count, descending source list, limit).
func TestPageViewRepositoryInsertAndOverview(t *testing.T) {
	db, _, cleanup := openPrimaryRepo(t)
	defer cleanup()
	if err := db.Exec(`TRUNCATE page_views RESTART IDENTITY`).Error; err != nil {
		t.Fatalf("truncate page_views: %v", err)
	}

	repo := primary.NewPageViewRepository(db)
	ctx := context.Background()
	base := time.Now().UTC()

	rows := []struct {
		path   string
		source string
		at     time.Time
	}{
		{"/", analytics.SourceDirect, base.Add(-1 * time.Hour)},
		{"/features", analytics.SourceDirect, base.Add(-2 * time.Hour)},
		{"/pricing", "www.baidu.com", base.Add(-3 * time.Hour)},
		{"/pricing", "www.baidu.com", base.Add(-8 * 24 * time.Hour)},
	}
	for _, row := range rows {
		view := &analytics.PageView{Path: row.path, Source: row.source, Locale: "en"}
		if err := repo.CreatePageView(ctx, view); err != nil {
			t.Fatalf("create page view: %v", err)
		}
		if view.ID == 0 || view.CreatedAt.IsZero() {
			t.Fatalf("create page view did not populate id/created_at: %+v", view)
		}
		if err := db.Exec(`UPDATE page_views SET created_at = ? WHERE id = ?`, row.at, view.ID).Error; err != nil {
			t.Fatalf("set created_at: %v", err)
		}
	}

	end := base.Add(time.Hour)

	pv, sourceCount, sources, err := repo.OverviewWindow(ctx, base.Add(-7*24*time.Hour), end, analytics.SourcesLimit)
	if err != nil {
		t.Fatalf("overview 7d: %v", err)
	}
	if pv != 3 {
		t.Fatalf("7d pv = %d, want 3", pv)
	}
	if sourceCount != 2 {
		t.Fatalf("7d source_count = %d, want 2", sourceCount)
	}
	if len(sources) != 2 || sources[0].Source != analytics.SourceDirect || sources[0].Count != 2 || sources[1].Source != "www.baidu.com" || sources[1].Count != 1 {
		t.Fatalf("7d sources = %+v, want [direct:2 www.baidu.com:1]", sources)
	}

	// The 30d window also includes the row from eight days ago.
	pv30, sourceCount30, _, err := repo.OverviewWindow(ctx, base.Add(-30*24*time.Hour), end, analytics.SourcesLimit)
	if err != nil {
		t.Fatalf("overview 30d: %v", err)
	}
	if pv30 != 4 || sourceCount30 != 2 {
		t.Fatalf("30d pv/source_count = %d/%d, want 4/2", pv30, sourceCount30)
	}

	// The source list is capped, but the distinct source count is not.
	pvLimited, sourceCountLimited, sourcesLimited, err := repo.OverviewWindow(ctx, base.Add(-7*24*time.Hour), end, 1)
	if err != nil {
		t.Fatalf("overview limited: %v", err)
	}
	if len(sourcesLimited) != 1 || pvLimited != 3 || sourceCountLimited != 2 {
		t.Fatalf("limited overview = pv %d, source_count %d, %d sources; want 3, 2, 1", pvLimited, sourceCountLimited, len(sourcesLimited))
	}
}
