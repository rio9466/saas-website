package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rio9466/easy-admin/server/internal/domain/analytics"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	"github.com/rio9466/easy-admin/server/internal/transport/http/middleware"
)

// stubAnalyticsService implements only the analytics methods under test; the
// embedded nil interface satisfies the rest of AnalyticsService.
type stubAnalyticsService struct {
	AnalyticsService
	recordErr error
	overview  *analytics.Overview
	rangeSeen string
}

func (s *stubAnalyticsService) RecordPageView(context.Context, analytics.PageViewInput, string) error {
	return s.recordErr
}

func (s *stubAnalyticsService) Overview(_ context.Context, _ Actor, rangeName string) (*analytics.Overview, error) {
	s.rangeSeen = rangeName
	return s.overview, nil
}

func TestPublicPageViewReturns200NoStore(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/public/page-view", NewAnalyticsPublicHandlers(&stubAnalyticsService{}).PageView)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/public/page-view",
		strings.NewReader(`{"path":"/features","referrer":"https://www.baidu.com/s"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	var env testEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Code != 0 {
		t.Fatalf("code = %d, want 0", env.Code)
	}
}

func TestPublicPageViewMalformedJSON(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/public/page-view", NewAnalyticsPublicHandlers(&stubAnalyticsService{}).PageView)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/public/page-view", strings.NewReader(`{"path":`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", w.Code, w.Body.String())
	}
	var env testEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Code != apperr.CodeValidation {
		t.Fatalf("code = %d, want %d", env.Code, apperr.CodeValidation)
	}
}

func TestPublicPageViewRateLimited(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	svc := &stubAnalyticsService{recordErr: apperr.New(http.StatusTooManyRequests, apperr.CodeRateLimited, "rate limited", apperr.ErrRateLimited)}
	r.POST("/api/v1/public/page-view", NewAnalyticsPublicHandlers(svc).PageView)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/public/page-view", strings.NewReader(`{"path":"/features"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429: %s", w.Code, w.Body.String())
	}
	var env testEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Code != apperr.CodeRateLimited {
		t.Fatalf("code = %d, want %d", env.Code, apperr.CodeRateLimited)
	}
}

func TestAdminAnalyticsOverviewShape(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	svc := &stubAnalyticsService{overview: &analytics.Overview{
		Range:       analytics.Range7d,
		PV:          12345,
		SourceCount: 2,
		Sources: []analytics.SourceCount{
			{Source: analytics.SourceDirect, Count: 8000},
			{Source: "www.baidu.com", Count: 4345},
		},
	}}
	h := NewAnalyticsAdminHandlers(svc)
	r.GET("/api/v1/admin/analytics/overview",
		func(c *gin.Context) { middleware.SetActor(c, 7, "s", "admin", "Admin", []string{"super_admin"}, nil) },
		middleware.RequirePermission(analytics.PermissionRead),
		h.Overview)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/analytics/overview?range=7d", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	if svc.rangeSeen != "7d" {
		t.Fatalf("range passed = %q, want 7d", svc.rangeSeen)
	}
	var env struct {
		Code int                   `json:"code"`
		Data analyticsOverviewData `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Code != 0 {
		t.Fatalf("code = %d, want 0", env.Code)
	}
	data := env.Data
	if data.Range != "7d" || data.PV != 12345 || data.SourceCount != 2 || len(data.Sources) != 2 {
		t.Fatalf("unexpected overview payload: %+v", data)
	}
	if data.Sources[0].Source != analytics.SourceDirect || data.Sources[0].Count != 8000 {
		t.Fatalf("unexpected first source: %+v", data.Sources[0])
	}
}

func TestAdminAnalyticsOverviewMissingPermission(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewAnalyticsAdminHandlers(&stubAnalyticsService{})
	r.GET("/api/v1/admin/analytics/overview",
		func(c *gin.Context) { middleware.SetActor(c, 7, "s", "viewer", "Viewer", []string{"viewer"}, nil) },
		middleware.RequirePermission(analytics.PermissionRead),
		h.Overview)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/analytics/overview", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403: %s", w.Code, w.Body.String())
	}
	var env testEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Code != apperr.CodeForbidden {
		t.Fatalf("code = %d, want %d", env.Code, apperr.CodeForbidden)
	}
}
