package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/rio9466/easy-admin/server/internal/transport/http/middleware"
	"github.com/rio9466/easy-admin/server/internal/transport/http/response"
)

// AnalyticsPublicHandlers serves the unauthenticated page-view ingest.
type AnalyticsPublicHandlers struct {
	svc AnalyticsService
}

// NewAnalyticsPublicHandlers constructs public analytics handlers.
func NewAnalyticsPublicHandlers(svc AnalyticsService) *AnalyticsPublicHandlers {
	return &AnalyticsPublicHandlers{svc: svc}
}

// PageView accepts one public page-view report. An invalid path is silently
// dropped by the service but still returns 200; the response is never cached.
func (h *AnalyticsPublicHandlers) PageView(c *gin.Context) {
	var req publicPageViewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationFailure(c)
		return
	}
	if err := h.svc.RecordPageView(c.Request.Context(), req.toInput(), c.ClientIP()); err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	response.OK(c, map[string]any{})
}

// AnalyticsAdminHandlers serves administrator analytics reads.
type AnalyticsAdminHandlers struct {
	svc AnalyticsService
}

// NewAnalyticsAdminHandlers constructs administrator analytics handlers.
func NewAnalyticsAdminHandlers(svc AnalyticsService) *AnalyticsAdminHandlers {
	return &AnalyticsAdminHandlers{svc: svc}
}

// Overview returns the aggregated page views and top sources for a range.
func (h *AnalyticsAdminHandlers) Overview(c *gin.Context) {
	view, err := h.svc.Overview(c.Request.Context(), actorFromContext(c), c.Query("range"))
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	response.OK(c, toAnalyticsOverviewData(view))
}
