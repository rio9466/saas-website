package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	userdomain "github.com/rio9466/easy-admin/server/internal/domain/user"
	"github.com/rio9466/easy-admin/server/internal/transport/http/middleware"
	"github.com/rio9466/easy-admin/server/internal/transport/http/response"
)

// UserLevelHandlers serves business-user level management.
type UserLevelHandlers struct {
	svc UserAdminService
}

// NewUserLevelHandlers constructs level handlers.
func NewUserLevelHandlers(svc UserAdminService) *UserLevelHandlers {
	return &UserLevelHandlers{svc: svc}
}

// List returns every user level (disabled included for historical context).
func (h *UserLevelHandlers) List(c *gin.Context) {
	levels, err := h.svc.ListUserLevels(c.Request.Context(), actorFromContext(c))
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, map[string]any{"items": toUserLevelList(levels)})
}

// Get returns one level.
func (h *UserLevelHandlers) Get(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	level, err := h.svc.GetUserLevel(c.Request.Context(), actorFromContext(c), id)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toUserLevelData(level))
}

// Create inserts a level.
func (h *UserLevelHandlers) Create(c *gin.Context) {
	var req createUserLevelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	threshold, err := userdomain.ParseDecimal4(req.ThresholdPoints)
	if err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	level, err := h.svc.CreateUserLevel(c.Request.Context(), actorFromContext(c), CreateLevelInput{
		Code:            strings.TrimSpace(req.Code),
		Name:            strings.TrimSpace(req.Name),
		IconURL:         strings.TrimSpace(req.IconURL),
		ThresholdPoints: threshold,
		SortOrder:       req.SortOrder,
		Enabled:         req.Enabled,
	})
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, toUserLevelData(level))
}

// Update selectively updates a level.
func (h *UserLevelHandlers) Update(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	var req updateUserLevelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	var threshold *userdomain.Decimal4
	if req.ThresholdPoints != nil {
		parsed, err := userdomain.ParseDecimal4(*req.ThresholdPoints)
		if err != nil {
			response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
			return
		}
		threshold = &parsed
	}
	level, err := h.svc.UpdateUserLevel(c.Request.Context(), actorFromContext(c), id, UpdateLevelInput{
		Name:            req.Name,
		IconURL:         req.IconURL,
		ThresholdPoints: threshold,
		SortOrder:       req.SortOrder,
		Enabled:         req.Enabled,
	})
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toUserLevelData(level))
}
