package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	"github.com/rio9466/easy-admin/server/internal/transport/http/middleware"
	"github.com/rio9466/easy-admin/server/internal/transport/http/response"
)

// ContentAdminHandlers serves administrator content CRUD.
type ContentAdminHandlers struct {
	svc ContentService
}

// NewContentAdminHandlers constructs content admin handlers.
func NewContentAdminHandlers(svc ContentService) *ContentAdminHandlers {
	return &ContentAdminHandlers{svc: svc}
}

func validationFailure(c *gin.Context) {
	response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
}

// --- site settings ---------------------------------------------------------

// GetSiteSettings returns the singleton with all translations.
func (h *ContentAdminHandlers) GetSiteSettings(c *gin.Context) {
	settings, err := h.svc.GetSiteSettings(c.Request.Context(), actorFromContext(c))
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toSiteSettingsData(settings))
}

// UpdateSiteSettings replaces the singleton under its optimistic lock.
func (h *ContentAdminHandlers) UpdateSiteSettings(c *gin.Context) {
	var req adminSiteSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationFailure(c)
		return
	}
	in, err := req.toInput()
	if err != nil {
		validationFailure(c)
		return
	}
	settings, err := h.svc.UpdateSiteSettings(c.Request.Context(), actorFromContext(c), in)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toSiteSettingsData(settings))
}

// --- navigation ------------------------------------------------------------

// ListNavigationItems returns a paginated navigation page.
func (h *ContentAdminHandlers) ListNavigationItems(c *gin.Context) {
	page, pageSize := pageParams(c)
	result, err := h.svc.ListNavigationItems(c.Request.Context(), actorFromContext(c), page, pageSize, c.Query("placement"))
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, pageData{Items: toNavigationItemList(result.Items), Total: result.Total, Page: result.Page, PageSize: result.PageSize})
}

// CreateNavigationItem inserts a navigation item.
func (h *ContentAdminHandlers) CreateNavigationItem(c *gin.Context) {
	var req adminNavigationItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationFailure(c)
		return
	}
	in, err := req.toInput()
	if err != nil {
		validationFailure(c)
		return
	}
	item, err := h.svc.CreateNavigationItem(c.Request.Context(), actorFromContext(c), in)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, toNavigationItemData(item))
}

// UpdateNavigationItem applies a partial navigation update.
func (h *ContentAdminHandlers) UpdateNavigationItem(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		validationFailure(c)
		return
	}
	var req adminNavigationItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationFailure(c)
		return
	}
	in, err := req.toInput()
	if err != nil {
		validationFailure(c)
		return
	}
	item, err := h.svc.UpdateNavigationItem(c.Request.Context(), actorFromContext(c), id, in)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toNavigationItemData(item))
}

// DeleteNavigationItem removes a navigation item.
func (h *ContentAdminHandlers) DeleteNavigationItem(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		validationFailure(c)
		return
	}
	if err := h.svc.DeleteNavigationItem(c.Request.Context(), actorFromContext(c), id); err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, map[string]any{})
}

// --- home sections ---------------------------------------------------------

// ListHomeSections returns a paginated home-section page.
func (h *ContentAdminHandlers) ListHomeSections(c *gin.Context) {
	page, pageSize := pageParams(c)
	result, err := h.svc.ListHomeSections(c.Request.Context(), actorFromContext(c), page, pageSize)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, pageData{Items: toHomeSectionList(result.Items), Total: result.Total, Page: result.Page, PageSize: result.PageSize})
}

// CreateHomeSection inserts a home section.
func (h *ContentAdminHandlers) CreateHomeSection(c *gin.Context) {
	var req adminHomeSectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationFailure(c)
		return
	}
	section, err := h.svc.CreateHomeSection(c.Request.Context(), actorFromContext(c), req.toInput())
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, toHomeSectionData(section))
}

// UpdateHomeSection applies a partial home-section update.
func (h *ContentAdminHandlers) UpdateHomeSection(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		validationFailure(c)
		return
	}
	var req adminHomeSectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationFailure(c)
		return
	}
	section, err := h.svc.UpdateHomeSection(c.Request.Context(), actorFromContext(c), id, req.toInput())
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toHomeSectionData(section))
}

// DeleteHomeSection removes a home section.
func (h *ContentAdminHandlers) DeleteHomeSection(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		validationFailure(c)
		return
	}
	if err := h.svc.DeleteHomeSection(c.Request.Context(), actorFromContext(c), id); err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, map[string]any{})
}

// --- features --------------------------------------------------------------

// ListFeatures returns a paginated feature page.
func (h *ContentAdminHandlers) ListFeatures(c *gin.Context) {
	page, pageSize := pageParams(c)
	result, err := h.svc.ListFeatures(c.Request.Context(), actorFromContext(c), page, pageSize)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, pageData{Items: toFeatureList(result.Items), Total: result.Total, Page: result.Page, PageSize: result.PageSize})
}

// CreateFeature inserts a feature.
func (h *ContentAdminHandlers) CreateFeature(c *gin.Context) {
	var req adminFeatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationFailure(c)
		return
	}
	feature, err := h.svc.CreateFeature(c.Request.Context(), actorFromContext(c), req.toInput())
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, toFeatureData(feature))
}

// UpdateFeature applies a partial feature update.
func (h *ContentAdminHandlers) UpdateFeature(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		validationFailure(c)
		return
	}
	var req adminFeatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationFailure(c)
		return
	}
	feature, err := h.svc.UpdateFeature(c.Request.Context(), actorFromContext(c), id, req.toInput())
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toFeatureData(feature))
}

// DeleteFeature removes a feature.
func (h *ContentAdminHandlers) DeleteFeature(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		validationFailure(c)
		return
	}
	if err := h.svc.DeleteFeature(c.Request.Context(), actorFromContext(c), id); err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, map[string]any{})
}

// --- pricing plans ---------------------------------------------------------

// ListPricingPlans returns a paginated pricing-plan page.
func (h *ContentAdminHandlers) ListPricingPlans(c *gin.Context) {
	page, pageSize := pageParams(c)
	result, err := h.svc.ListPricingPlans(c.Request.Context(), actorFromContext(c), page, pageSize)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, pageData{Items: toPricingPlanList(result.Items), Total: result.Total, Page: result.Page, PageSize: result.PageSize})
}

// CreatePricingPlan inserts a plan.
func (h *ContentAdminHandlers) CreatePricingPlan(c *gin.Context) {
	var req adminPricingPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationFailure(c)
		return
	}
	plan, err := h.svc.CreatePricingPlan(c.Request.Context(), actorFromContext(c), req.toInput())
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, toPricingPlanData(plan))
}

// UpdatePricingPlan applies a partial plan update.
func (h *ContentAdminHandlers) UpdatePricingPlan(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		validationFailure(c)
		return
	}
	var req adminPricingPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationFailure(c)
		return
	}
	plan, err := h.svc.UpdatePricingPlan(c.Request.Context(), actorFromContext(c), id, req.toInput())
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toPricingPlanData(plan))
}

// DeletePricingPlan removes a plan.
func (h *ContentAdminHandlers) DeletePricingPlan(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		validationFailure(c)
		return
	}
	if err := h.svc.DeletePricingPlan(c.Request.Context(), actorFromContext(c), id); err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, map[string]any{})
}

// --- pages -----------------------------------------------------------------

// ListPages returns a paginated page listing.
func (h *ContentAdminHandlers) ListPages(c *gin.Context) {
	page, pageSize := pageParams(c)
	result, err := h.svc.ListPages(c.Request.Context(), actorFromContext(c), page, pageSize)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, pageData{Items: toPageList(result.Items), Total: result.Total, Page: result.Page, PageSize: result.PageSize})
}

// GetPage returns one page with all translations.
func (h *ContentAdminHandlers) GetPage(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		validationFailure(c)
		return
	}
	page, err := h.svc.GetPage(c.Request.Context(), actorFromContext(c), id)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toPageData(page))
}

// CreatePage inserts a page.
func (h *ContentAdminHandlers) CreatePage(c *gin.Context) {
	var req adminPageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationFailure(c)
		return
	}
	page, err := h.svc.CreatePage(c.Request.Context(), actorFromContext(c), req.toInput())
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, toPageData(page))
}

// UpdatePage applies a partial page update.
func (h *ContentAdminHandlers) UpdatePage(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		validationFailure(c)
		return
	}
	var req adminPageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationFailure(c)
		return
	}
	page, err := h.svc.UpdatePage(c.Request.Context(), actorFromContext(c), id, req.toInput())
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toPageData(page))
}

// DeletePage removes a page.
func (h *ContentAdminHandlers) DeletePage(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		validationFailure(c)
		return
	}
	if err := h.svc.DeletePage(c.Request.Context(), actorFromContext(c), id); err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, map[string]any{})
}

// --- doc categories --------------------------------------------------------

// ListDocCategories returns a paginated category listing.
func (h *ContentAdminHandlers) ListDocCategories(c *gin.Context) {
	page, pageSize := pageParams(c)
	result, err := h.svc.ListDocCategories(c.Request.Context(), actorFromContext(c), page, pageSize)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, pageData{Items: toDocCategoryList(result.Items), Total: result.Total, Page: result.Page, PageSize: result.PageSize})
}

// CreateDocCategory inserts a documentation category.
func (h *ContentAdminHandlers) CreateDocCategory(c *gin.Context) {
	var req adminDocCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationFailure(c)
		return
	}
	category, err := h.svc.CreateDocCategory(c.Request.Context(), actorFromContext(c), req.toInput())
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, toDocCategoryData(category))
}

// UpdateDocCategory applies a partial category update.
func (h *ContentAdminHandlers) UpdateDocCategory(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		validationFailure(c)
		return
	}
	var req adminDocCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationFailure(c)
		return
	}
	category, err := h.svc.UpdateDocCategory(c.Request.Context(), actorFromContext(c), id, req.toInput())
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toDocCategoryData(category))
}

// DeleteDocCategory removes a category.
func (h *ContentAdminHandlers) DeleteDocCategory(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		validationFailure(c)
		return
	}
	if err := h.svc.DeleteDocCategory(c.Request.Context(), actorFromContext(c), id); err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, map[string]any{})
}

// --- doc articles ----------------------------------------------------------

// ListDocArticles returns a paginated article listing.
func (h *ContentAdminHandlers) ListDocArticles(c *gin.Context) {
	page, pageSize := pageParams(c)
	result, err := h.svc.ListDocArticles(c.Request.Context(), actorFromContext(c), page, pageSize)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, pageData{Items: toDocArticleList(result.Items), Total: result.Total, Page: result.Page, PageSize: result.PageSize})
}

// GetDocArticle returns one article with all translations.
func (h *ContentAdminHandlers) GetDocArticle(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		validationFailure(c)
		return
	}
	article, err := h.svc.GetDocArticle(c.Request.Context(), actorFromContext(c), id)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toDocArticleData(article))
}

// CreateDocArticle inserts a documentation article.
func (h *ContentAdminHandlers) CreateDocArticle(c *gin.Context) {
	var req adminDocArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationFailure(c)
		return
	}
	in, err := req.toInput()
	if err != nil {
		validationFailure(c)
		return
	}
	article, err := h.svc.CreateDocArticle(c.Request.Context(), actorFromContext(c), in)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, toDocArticleData(article))
}

// UpdateDocArticle applies a partial article update.
func (h *ContentAdminHandlers) UpdateDocArticle(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		validationFailure(c)
		return
	}
	var req adminDocArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationFailure(c)
		return
	}
	in, err := req.toInput()
	if err != nil {
		validationFailure(c)
		return
	}
	article, err := h.svc.UpdateDocArticle(c.Request.Context(), actorFromContext(c), id, in)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toDocArticleData(article))
}

// DeleteDocArticle removes an article.
func (h *ContentAdminHandlers) DeleteDocArticle(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		validationFailure(c)
		return
	}
	if err := h.svc.DeleteDocArticle(c.Request.Context(), actorFromContext(c), id); err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, map[string]any{})
}
