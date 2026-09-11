package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	"github.com/rio9466/easy-admin/server/internal/transport/http/middleware"
	"github.com/rio9466/easy-admin/server/internal/transport/http/response"
)

// RoleHandlers serves role and permission catalog endpoints.
type RoleHandlers struct {
	svc AdminAuthService
}

// NewRoleHandlers constructs role handlers.
func NewRoleHandlers(svc AdminAuthService) *RoleHandlers {
	return &RoleHandlers{svc: svc}
}

// ListRoles lists roles.
func (h *RoleHandlers) ListRoles(c *gin.Context) {
	roles, err := h.svc.ListRoles(c.Request.Context(), actorFromContext(c))
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, map[string]any{"items": toRoleList(roles)})
}

// GetRole returns one role.
func (h *RoleHandlers) GetRole(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	role, err := h.svc.GetRole(c.Request.Context(), actorFromContext(c), id)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toRoleData(role))
}

// CreateRole creates a custom role.
func (h *RoleHandlers) CreateRole(c *gin.Context) {
	var req createRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	req.Name = strings.TrimSpace(req.Name)
	if req.Code == "" || req.Name == "" {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	role, err := h.svc.CreateRole(c.Request.Context(), actorFromContext(c), CreateRoleInput{
		Code:        req.Code,
		Name:        req.Name,
		Description: strings.TrimSpace(req.Description),
	})
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, toRoleData(role))
}

// UpdateRole patches a role.
func (h *RoleHandlers) UpdateRole(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req updateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	role, err := h.svc.UpdateRole(c.Request.Context(), actorFromContext(c), id, UpdateRoleInput{
		Name:        req.Name,
		Description: req.Description,
		Enabled:     req.Enabled,
	})
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toRoleData(role))
}

// ReplacePermissions replaces a role's permission set.
func (h *RoleHandlers) ReplacePermissions(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var req replacePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	role, err := h.svc.ReplaceRolePermissions(c.Request.Context(), actorFromContext(c), id, req.PermissionCodes)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toRoleData(role))
}

// ListPermissions returns the permission catalog.
func (h *RoleHandlers) ListPermissions(c *gin.Context) {
	perms, err := h.svc.ListPermissions(c.Request.Context(), actorFromContext(c))
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, map[string]any{"items": toPermissionList(perms)})
}
