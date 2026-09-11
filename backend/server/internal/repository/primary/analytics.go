package primary

import (
	"context"
	"fmt"
	"time"

	"github.com/rio9466/easy-admin/server/internal/domain/analytics"
	"gorm.io/gorm"
)

// PageViewRepository persists and aggregates page views in primary PostgreSQL.
type PageViewRepository struct {
	db *gorm.DB
}

// NewPageViewRepository constructs a PageViewRepository backed by db.
func NewPageViewRepository(db *gorm.DB) *PageViewRepository {
	return &PageViewRepository{db: db}
}

func (r *PageViewRepository) session(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

// CreatePageView inserts one page-view row and populates its id and time.
func (r *PageViewRepository) CreatePageView(ctx context.Context, in *analytics.PageView) error {
	if in == nil {
		return fmt.Errorf("page view is nil")
	}
	now := time.Now().UTC()
	model := pageViewModel{
		Path:      in.Path,
		Source:    in.Source,
		Locale:    in.Locale,
		CreatedAt: now,
	}
	if err := r.session(ctx).Create(&model).Error; err != nil {
		return wrapDBError(err)
	}
	in.ID = model.ID
	in.CreatedAt = now
	return nil
}

// OverviewWindow returns the page-view total, the number of distinct sources,
// and the top sources by count descending for the inclusive [start, end]
// window. limit caps the returned source list; a non-positive limit uses the
// contract default.
func (r *PageViewRepository) OverviewWindow(ctx context.Context, start, end time.Time, limit int) (int64, int64, []analytics.SourceCount, error) {
	if limit <= 0 {
		limit = analytics.SourcesLimit
	}

	var totals struct {
		PV          int64 `gorm:"column:pv"`
		SourceCount int64 `gorm:"column:source_count"`
	}
	if err := r.session(ctx).Model(&pageViewModel{}).
		Select("count(*) AS pv, count(DISTINCT source) AS source_count").
		Where("created_at >= ? AND created_at <= ?", start, end).
		Scan(&totals).Error; err != nil {
		return 0, 0, nil, wrapDBError(err)
	}

	var rows []struct {
		Source string `gorm:"column:source"`
		Count  int64  `gorm:"column:count"`
	}
	if err := r.session(ctx).Model(&pageViewModel{}).
		Select("source, count(*) AS count").
		Where("created_at >= ? AND created_at <= ?", start, end).
		Group("source").
		Order("count DESC, source ASC").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return 0, 0, nil, wrapDBError(err)
	}

	sources := make([]analytics.SourceCount, 0, len(rows))
	for _, row := range rows {
		sources = append(sources, analytics.SourceCount{Source: row.Source, Count: row.Count})
	}
	return totals.PV, totals.SourceCount, sources, nil
}
