package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	"github.com/rio9466/easy-admin/server/internal/transport/http/middleware"
	"github.com/rio9466/easy-admin/server/internal/transport/http/response"
)

// AuditHandlers serves read-only audit event endpoints.
type AuditHandlers struct {
	svc AdminAuthService
}

// NewAuditHandlers constructs audit handlers.
func NewAuditHandlers(svc AdminAuthService) *AuditHandlers {
	return &AuditHandlers{svc: svc}
}

// List returns a filtered, paginated audit event list.
func (h *AuditHandlers) List(c *gin.Context) {
	page, pageSize := parsePagination(c)
	in := AuditListInput{
		Action:       strings.TrimSpace(c.Query("action")),
		ResourceType: strings.TrimSpace(c.Query("resource_type")),
		Outcome:      strings.TrimSpace(c.Query("outcome")),
		RequestID:    strings.TrimSpace(c.Query("request_id")),
		Page:         page,
		PageSize:     pageSize,
	}

	if raw := strings.TrimSpace(c.Query("actor_id")); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
			return
		}
		in.ActorID = &id
	}
	from, ok := parseOptionalRFC3339(c, "from")
	if !ok {
		return
	}
	to, ok := parseOptionalRFC3339(c, "to")
	if !ok {
		return
	}
	in.From = from
	in.To = to

	result, err := h.svc.ListAuditEvents(c.Request.Context(), actorFromContext(c), in)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, pageData{
		Items:    toAuditEventList(result.Items),
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

// Get returns one audit event.
func (h *AuditHandlers) Get(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	event, err := h.svc.GetAuditEvent(c.Request.Context(), actorFromContext(c), id)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toAuditEventData(event))
}

func parseOptionalRFC3339(c *gin.Context, name string) (*time.Time, bool) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return nil, true
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return nil, false
	}
	utc := t.UTC()
	return &utc, true
}
