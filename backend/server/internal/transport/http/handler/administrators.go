package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	"github.com/rio9466/easy-admin/server/internal/transport/http/middleware"
	"github.com/rio9466/easy-admin/server/internal/transport/http/response"
)

// AdministratorHandlers serves administrator management endpoints.
type AdministratorHandlers struct {
	svc AdminAuthService
}

// NewAdministratorHandlers constructs administrator handlers.
func NewAdministratorHandlers(svc AdminAuthService) *AdministratorHandlers {
	return &AdministratorHandlers{svc: svc}
}

// List returns a paginated administrator list.
func (h *AdministratorHandlers) List(c *gin.Context) {
	page, pageSize := parsePagination(c)
	query := strings.TrimSpace(c.Query("q"))
	result, err := h.svc.ListAdministrators(c.Request.Context(), actorFromContext(c), page, pageSize, query)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, pageData{
		Items:    toAdministratorList(result.Items),
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

// Get returns one administrator.
func (h *AdministratorHandlers) Get(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	admin, err := h.svc.GetAdministrator(c.Request.Context(), actorFromContext(c), id)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toAdministratorData(admin))
}

// Create creates an administrator.
func (h *AdministratorHandlers) Create(c *gin.Context) {
	var req createAdministratorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	admin, err := h.svc.CreateAdministrator(c.Request.Context(), actorFromContext(c), CreateAdministratorInput{
		Username:    req.Username,
		Password:    req.Password,
		DisplayName: strings.TrimSpace(req.DisplayName),
		RoleCodes:   req.RoleCodes,
		Enabled:     true,
	})
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, toAdministratorData(admin))
}

// Update patches an administrator profile.
func (h *AdministratorHandlers) Update(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req updateAdministratorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	admin, err := h.svc.UpdateAdministrator(c.Request.Context(), actorFromContext(c), id, UpdateAdministratorInput{
		DisplayName: req.DisplayName,
	})
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toAdministratorData(admin))
}

// Enable enables an administrator.
func (h *AdministratorHandlers) Enable(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	admin, err := h.svc.EnableAdministrator(c.Request.Context(), actorFromContext(c), id)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toAdministratorData(admin))
}

// Disable disables an administrator.
func (h *AdministratorHandlers) Disable(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	admin, err := h.svc.DisableAdministrator(c.Request.Context(), actorFromContext(c), id)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toAdministratorData(admin))
}

// ResetPassword resets another administrator's password.
func (h *AdministratorHandlers) ResetPassword(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	if req.NewPassword == "" {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	if err := h.svc.ResetAdministratorPassword(c.Request.Context(), actorFromContext(c), id, req.NewPassword); err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, map[string]any{})
}

// AssignRoles replaces an administrator's role assignments.
func (h *AdministratorHandlers) AssignRoles(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req assignRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	admin, err := h.svc.AssignAdministratorRoles(c.Request.Context(), actorFromContext(c), id, req.RoleCodes)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toAdministratorData(admin))
}

func parsePathID(c *gin.Context, param string) (int64, bool) {
	raw := strings.TrimSpace(c.Param(param))
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return 0, false
	}
	return id, true
}

func parsePagination(c *gin.Context) (page, pageSize int) {
	page = 1
	pageSize = 20
	if v := strings.TrimSpace(c.Query("page")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := strings.TrimSpace(c.Query("page_size")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			pageSize = n
		}
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
