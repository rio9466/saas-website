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

// UserAdminHandlers serves administrator business-user management.
type UserAdminHandlers struct {
	svc UserAdminService
}

// NewUserAdminHandlers constructs business-user admin handlers.
func NewUserAdminHandlers(svc UserAdminService) *UserAdminHandlers {
	return &UserAdminHandlers{svc: svc}
}

// List returns a paginated business-user page.
func (h *UserAdminHandlers) List(c *gin.Context) {
	page, pageSize := pageParams(c)
	filter := UserListInput{Query: c.Query("q"), Status: c.Query("status")}
	if raw := c.Query("level_id"); raw != "" {
		if id, err := parseID(raw); err == nil && id > 0 {
			filter.LevelID = &id
		}
	}
	result, err := h.svc.ListUsers(c.Request.Context(), actorFromContext(c), page, pageSize, filter)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, pageData{
		Items:    toBusinessUserList(result.Items),
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

// Get returns one business user by id.
func (h *UserAdminHandlers) Get(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	user, err := h.svc.GetUser(c.Request.Context(), actorFromContext(c), id)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toBusinessUserData(user))
}

// Create creates an active, email-verified business user.
func (h *UserAdminHandlers) Create(c *gin.Context) {
	var req createBusinessUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	if req.Username == "" || req.Email == "" || req.Password == "" {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	user, err := h.svc.CreateUser(c.Request.Context(), actorFromContext(c), CreateUserInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Nickname: req.Nickname,
	})
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, toBusinessUserData(user))
}

// Update updates email/nickname/avatar/remark only.
func (h *UserAdminHandlers) Update(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	var req updateBusinessUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	user, err := h.svc.UpdateUser(c.Request.Context(), actorFromContext(c), id, UpdateUserInput{
		Email:     req.Email,
		Nickname:  req.Nickname,
		AvatarURL: req.AvatarURL,
		Remark:    req.Remark,
	})
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toBusinessUserData(user))
}

// Enable re-enables a disabled business user.
func (h *UserAdminHandlers) Enable(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	user, err := h.svc.EnableUser(c.Request.Context(), actorFromContext(c), id)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toBusinessUserData(user))
}

// Disable disables an active business user.
func (h *UserAdminHandlers) Disable(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	user, err := h.svc.DisableUser(c.Request.Context(), actorFromContext(c), id)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toBusinessUserData(user))
}

// ResetPassword sets a new password and invalidates sessions.
func (h *UserAdminHandlers) ResetPassword(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	var req resetBusinessUserPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	if req.NewPassword == "" {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	if err := h.svc.ResetUserPassword(c.Request.Context(), actorFromContext(c), id, req.NewPassword); err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, map[string]any{})
}

// AdjustPoints applies the atomic ledger adjustment.
func (h *UserAdminHandlers) AdjustPoints(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	var req adjustPointsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	pointsDelta, err := userdomain.ParseDecimal4(req.PointsDelta)
	if err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	consumptionDelta, err := userdomain.ParseDecimal4(req.ConsumptionDelta)
	if err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	if strings.TrimSpace(req.Reason) == "" || strings.TrimSpace(req.IdempotencyKey) == "" {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	pt, err := h.svc.AdjustPoints(c.Request.Context(), actorFromContext(c), id, AdjustPointsInput{
		PointsDelta:      pointsDelta,
		ConsumptionDelta: consumptionDelta,
		Reason:           strings.TrimSpace(req.Reason),
		IdempotencyKey:   strings.TrimSpace(req.IdempotencyKey),
	})
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toPointTransactionData(pt))
}

// ListPointTransactions returns the immutable ledger page for a user.
func (h *UserAdminHandlers) ListPointTransactions(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	page, pageSize := pageParams(c)
	result, err := h.svc.ListPointTransactions(c.Request.Context(), actorFromContext(c), id, page, pageSize)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, pageData{
		Items:    toPointTransactionList(result.Items),
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

// AssignLevel sets manual or automatic level mode.
func (h *UserAdminHandlers) AssignLevel(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	var req assignBusinessUserLevelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	mode := strings.TrimSpace(req.LevelMode)
	levelID := int64(0)
	if req.LevelID != "" {
		levelID, err = parseID(req.LevelID)
		if err != nil || levelID <= 0 {
			response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
			return
		}
	}
	user, err := h.svc.AssignUserLevel(c.Request.Context(), actorFromContext(c), AssignLevelInput{
		UserID:  id,
		LevelID: levelID,
		Mode:    mode,
	})
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toBusinessUserData(user))
}

func pageParams(c *gin.Context) (int, int) {
	page := 1
	if raw := c.Query("page"); raw != "" {
		if v, err := parseID(raw); err == nil && v > 0 && v <= 100000 {
			page = int(v)
		}
	}
	pageSize := 20
	if raw := c.Query("page_size"); raw != "" {
		if v, err := parseID(raw); err == nil && v > 0 && v <= 100 {
			pageSize = int(v)
		}
	}
	return page, pageSize
}
