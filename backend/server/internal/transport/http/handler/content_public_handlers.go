package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/rio9466/easy-admin/server/internal/transport/http/middleware"
	"github.com/rio9466/easy-admin/server/internal/transport/http/response"
)

// publicContentCacheControl is the shared caching directive for public content
// (contract §4).
const publicContentCacheControl = "public, max-age=60, stale-while-revalidate=300"

// setPublicCache applies the public content cache header.
func setPublicCache(c *gin.Context) {
	c.Header("Cache-Control", publicContentCacheControl)
}

// ContentPublicHandlers serves the unauthenticated public content endpoints.
type ContentPublicHandlers struct {
	svc ContentService
}

// NewContentPublicHandlers constructs public content handlers.
func NewContentPublicHandlers(svc ContentService) *ContentPublicHandlers {
	return &ContentPublicHandlers{svc: svc}
}

// Navigation returns the visible navigation tree.
func (h *ContentPublicHandlers) Navigation(c *gin.Context) {
	view, err := h.svc.GetPublicNavigation(c.Request.Context(), c.Query("locale"), c.Query("placement"))
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	setPublicCache(c)
	response.OK(c, view)
}

// Home returns published home sections.
func (h *ContentPublicHandlers) Home(c *gin.Context) {
	view, err := h.svc.GetPublicHome(c.Request.Context(), c.Query("locale"))
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	setPublicCache(c)
	response.OK(c, view)
}

// Features returns published features.
func (h *ContentPublicHandlers) Features(c *gin.Context) {
	view, err := h.svc.GetPublicFeatures(c.Request.Context(), c.Query("locale"))
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	setPublicCache(c)
	response.OK(c, view)
}

// Pricing returns visible pricing plans.
func (h *ContentPublicHandlers) Pricing(c *gin.Context) {
	view, err := h.svc.GetPublicPricing(c.Request.Context(), c.Query("locale"))
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	setPublicCache(c)
	response.OK(c, view)
}

// Pages returns published pages.
func (h *ContentPublicHandlers) Pages(c *gin.Context) {
	view, err := h.svc.GetPublicPages(c.Request.Context(), c.Query("locale"))
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	setPublicCache(c)
	response.OK(c, view)
}

// Page returns one published page by slug.
func (h *ContentPublicHandlers) Page(c *gin.Context) {
	view, err := h.svc.GetPublicPage(c.Request.Context(), c.Query("locale"), c.Param("slug"))
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	setPublicCache(c)
	response.OK(c, view)
}

// Docs returns the documentation tree.
func (h *ContentPublicHandlers) Docs(c *gin.Context) {
	view, err := h.svc.GetPublicDocs(c.Request.Context(), c.Query("locale"))
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	setPublicCache(c)
	response.OK(c, view)
}

// Doc returns one published article by slug.
func (h *ContentPublicHandlers) Doc(c *gin.Context) {
	view, err := h.svc.GetPublicDoc(c.Request.Context(), c.Query("locale"), c.Param("slug"))
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	setPublicCache(c)
	response.OK(c, view)
}
