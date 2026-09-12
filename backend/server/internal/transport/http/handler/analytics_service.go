package handler

import (
	"context"

	"github.com/rio9466/easy-admin/server/internal/domain/analytics"
	analyticssvc "github.com/rio9466/easy-admin/server/internal/service/analyticssvc"
)

// AnalyticsService is the transport-facing analytics surface: public page-view
// ingest plus the administrator overview. Input types are shared with the
// service package.
type AnalyticsService interface {
	RecordPageView(ctx context.Context, in analytics.PageViewInput, sourceIP string) error
	Overview(ctx context.Context, actor Actor, rangeName string) (*analytics.Overview, error)
}

// AnalyticsServiceAdapter adapts *analyticssvc.Service to AnalyticsService.
type AnalyticsServiceAdapter struct {
	Svc *analyticssvc.Service
}

var _ AnalyticsService = (*AnalyticsServiceAdapter)(nil)

func (a *AnalyticsServiceAdapter) RecordPageView(ctx context.Context, in analytics.PageViewInput, sourceIP string) error {
	return a.Svc.RecordPageView(ctx, in, sourceIP)
}

func (a *AnalyticsServiceAdapter) Overview(ctx context.Context, actor Actor, rangeName string) (*analytics.Overview, error) {
	return a.Svc.Overview(ctx, toAnalyticsActor(actor), rangeName)
}

func toAnalyticsActor(a Actor) analyticssvc.Actor {
	return analyticssvc.Actor{
		AdminID:          a.ID,
		AdminUsername:    a.Username,
		AdminDisplayName: a.DisplayName,
		AdminRoleCodes:   a.RoleCodes,
		RequestID:        a.RequestID,
		SourceIP:         a.SourceIP,
		UserAgent:        a.UserAgent,
	}
}
