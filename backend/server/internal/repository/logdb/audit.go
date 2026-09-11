package logdb

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"gorm.io/gorm"
)

// AuditListFilter selects and paginates audit events.
type AuditListFilter struct {
	ActorID      *int64
	Action       string
	ResourceType string
	Outcome      string
	RequestID    string
	From         *time.Time
	To           *time.Time
	Page         int
	PageSize     int
}

// AuditRepository persists audit events in the log PostgreSQL database.
type AuditRepository struct {
	db *gorm.DB
}

// NewAuditRepository constructs an AuditRepository backed by db.
func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) session(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

type auditEventModel struct {
	ID               int64           `gorm:"column:id;primaryKey"`
	ActorID          *int64          `gorm:"column:actor_id"`
	ActorUsername    string          `gorm:"column:actor_username"`
	ActorDisplayName string          `gorm:"column:actor_display_name"`
	ActorRoleCodes   textArray       `gorm:"column:actor_role_codes;type:text[]"`
	Action           string          `gorm:"column:action"`
	ResourceType     string          `gorm:"column:resource_type"`
	ResourceID       string          `gorm:"column:resource_id"`
	Outcome          string          `gorm:"column:outcome"`
	Details          json.RawMessage `gorm:"column:details;type:jsonb"`
	RequestID        string          `gorm:"column:request_id"`
	SourceIP         string          `gorm:"column:source_ip"`
	UserAgent        string          `gorm:"column:user_agent"`
	EventAt          time.Time       `gorm:"column:event_at"`
	CreatedAt        time.Time       `gorm:"column:created_at"`
}

func (auditEventModel) TableName() string { return "audit_events" }

// CreatePending inserts a pending audit event and returns its ID.
func (r *AuditRepository) CreatePending(ctx context.Context, event *adminauth.AuditEvent) (int64, error) {
	if event == nil {
		return 0, fmt.Errorf("audit event is nil")
	}

	details, err := marshalDetails(event.Details)
	if err != nil {
		return 0, err
	}

	now := time.Now().UTC()
	eventAt := event.EventAt
	if eventAt.IsZero() {
		eventAt = now
	}

	model := auditEventModel{
		ActorID:          event.ActorID,
		ActorUsername:    event.ActorUsername,
		ActorDisplayName: event.ActorDisplayName,
		ActorRoleCodes:   textArray(cloneStrings(event.ActorRoleCodes)),
		Action:           event.Action,
		ResourceType:     event.ResourceType,
		ResourceID:       event.ResourceID,
		Outcome:          adminauth.AuditOutcomePending,
		Details:          details,
		RequestID:        event.RequestID,
		SourceIP:         event.SourceIP,
		UserAgent:        event.UserAgent,
		EventAt:          eventAt.UTC(),
		CreatedAt:        now,
	}

	if err := r.session(ctx).Create(&model).Error; err != nil {
		return 0, wrapDBError(err)
	}

	event.ID = model.ID
	event.Outcome = model.Outcome
	event.EventAt = model.EventAt
	event.CreatedAt = model.CreatedAt
	return model.ID, nil
}

// Finalize marks a pending event with its final resource ID, outcome, and
// details. Creates set the resource ID here because the row ID is only known
// after the primary insert.
func (r *AuditRepository) Finalize(ctx context.Context, id int64, resourceID string, outcome string, details map[string]any) error {
	payload, err := marshalDetails(details)
	if err != nil {
		return err
	}

	res := r.session(ctx).Model(&auditEventModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"resource_id": resourceID,
			"outcome":     outcome,
			"details":     payload,
		})
	if res.Error != nil {
		return wrapDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// GetByID loads a single audit event.
func (r *AuditRepository) GetByID(ctx context.Context, id int64) (*adminauth.AuditEvent, error) {
	var model auditEventModel
	if err := r.session(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, wrapDBError(err)
	}
	return toDomainAuditEvent(model)
}

// List returns a filtered, paginated audit event page ordered by event_at descending.
func (r *AuditRepository) List(ctx context.Context, filter AuditListFilter) (adminauth.Page[adminauth.AuditEvent], error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	out := adminauth.Page[adminauth.AuditEvent]{Page: page, PageSize: pageSize, Items: []adminauth.AuditEvent{}}

	db := r.session(ctx).Model(&auditEventModel{})
	if filter.ActorID != nil {
		db = db.Where("actor_id = ?", *filter.ActorID)
	}
	if action := strings.TrimSpace(filter.Action); action != "" {
		db = db.Where("action = ?", action)
	}
	if resourceType := strings.TrimSpace(filter.ResourceType); resourceType != "" {
		db = db.Where("resource_type = ?", resourceType)
	}
	if outcome := strings.TrimSpace(filter.Outcome); outcome != "" {
		db = db.Where("outcome = ?", outcome)
	}
	if requestID := strings.TrimSpace(filter.RequestID); requestID != "" {
		db = db.Where("request_id = ?", requestID)
	}
	if filter.From != nil {
		db = db.Where("event_at >= ?", filter.From.UTC())
	}
	if filter.To != nil {
		db = db.Where("event_at <= ?", filter.To.UTC())
	}

	if err := db.Count(&out.Total).Error; err != nil {
		return out, wrapDBError(err)
	}

	var models []auditEventModel
	err := db.Order("event_at DESC, id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&models).Error
	if err != nil {
		return out, wrapDBError(err)
	}

	out.Items = make([]adminauth.AuditEvent, 0, len(models))
	for _, model := range models {
		event, err := toDomainAuditEvent(model)
		if err != nil {
			return out, err
		}
		out.Items = append(out.Items, *event)
	}
	return out, nil
}

func toDomainAuditEvent(model auditEventModel) (*adminauth.AuditEvent, error) {
	details, err := unmarshalDetails(model.Details)
	if err != nil {
		return nil, err
	}
	return &adminauth.AuditEvent{
		ID:               model.ID,
		ActorID:          model.ActorID,
		ActorUsername:    model.ActorUsername,
		ActorDisplayName: model.ActorDisplayName,
		ActorRoleCodes:   cloneStrings([]string(model.ActorRoleCodes)),
		Action:           model.Action,
		ResourceType:     model.ResourceType,
		ResourceID:       model.ResourceID,
		Outcome:          model.Outcome,
		Details:          details,
		RequestID:        model.RequestID,
		SourceIP:         model.SourceIP,
		UserAgent:        model.UserAgent,
		EventAt:          model.EventAt,
		CreatedAt:        model.CreatedAt,
	}, nil
}

func marshalDetails(details map[string]any) (json.RawMessage, error) {
	if details == nil {
		return json.RawMessage(`{}`), nil
	}
	b, err := json.Marshal(details)
	if err != nil {
		return nil, fmt.Errorf("marshal audit details: %w", err)
	}
	return json.RawMessage(b), nil
}

func unmarshalDetails(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var details map[string]any
	if err := json.Unmarshal(raw, &details); err != nil {
		return nil, fmt.Errorf("unmarshal audit details: %w", err)
	}
	if details == nil {
		details = map[string]any{}
	}
	return details, nil
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func cloneStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

// textArray maps PostgreSQL text[] values through database/sql.
type textArray []string

func (a textArray) Value() (driver.Value, error) {
	if a == nil {
		return "{}", nil
	}
	var b strings.Builder
	b.WriteByte('{')
	for i, s := range a {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('"')
		b.WriteString(escapeArrayElement(s))
		b.WriteByte('"')
	}
	b.WriteByte('}')
	return b.String(), nil
}

func (a *textArray) Scan(src any) error {
	if src == nil {
		*a = []string{}
		return nil
	}
	var raw string
	switch v := src.(type) {
	case string:
		raw = v
	case []byte:
		raw = string(v)
	default:
		return fmt.Errorf("unsupported text[] scan type %T", src)
	}
	parsed, err := parseTextArray(raw)
	if err != nil {
		return err
	}
	*a = parsed
	return nil
}

func escapeArrayElement(s string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return replacer.Replace(s)
}

func parseTextArray(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "{}" {
		return []string{}, nil
	}
	if len(s) < 2 || s[0] != '{' || s[len(s)-1] != '}' {
		return nil, fmt.Errorf("invalid text[] value")
	}
	inner := s[1 : len(s)-1]
	if inner == "" {
		return []string{}, nil
	}

	var (
		out      []string
		current  strings.Builder
		inQuotes bool
		escape   bool
	)
	flush := func() {
		out = append(out, current.String())
		current.Reset()
	}

	for i := 0; i < len(inner); i++ {
		ch := inner[i]
		if escape {
			current.WriteByte(ch)
			escape = false
			continue
		}
		if ch == '\\' && inQuotes {
			escape = true
			continue
		}
		if ch == '"' {
			inQuotes = !inQuotes
			continue
		}
		if ch == ',' && !inQuotes {
			flush()
			continue
		}
		current.WriteByte(ch)
	}
	if inQuotes || escape {
		return nil, fmt.Errorf("invalid text[] value")
	}
	flush()
	return out, nil
}
