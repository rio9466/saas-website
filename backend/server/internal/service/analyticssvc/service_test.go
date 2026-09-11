package analyticssvc

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/rio9466/easy-admin/server/internal/domain/analytics"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
)

// TestRecordPageViewDropsInvalidPath proves an invalid report is a silent
// success: it returns before touching the repository or the rate limiter.
func TestRecordPageViewDropsInvalidPath(t *testing.T) {
	t.Parallel()
	svc := &Service{views: primary.NewPageViewRepository(nil), logger: slog.Default()}
	for _, path := range []string{"", "features", "https://example.com/x", "/" + strings.Repeat("a", analytics.MaxPathLength)} {
		if err := svc.RecordPageView(context.Background(), analytics.PageViewInput{Path: path}, "127.0.0.1"); err != nil {
			t.Fatalf("RecordPageView(%q) = %v, want nil silent drop", path, err)
		}
	}
}
