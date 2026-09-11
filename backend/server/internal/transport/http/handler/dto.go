package handler

import (
	"strconv"
	"time"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
)

// --- request DTOs ---

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// updateMeRequest only accepts display_name for the current administrator.
// Administrator ID, username, roles, permissions, and enabled state are not
// accepted by design (the request DTO never binds them).
type updateMeRequest struct {
	DisplayName *string `json:"display_name"`
}

type createAdministratorRequest struct {
	Username    string   `json:"username"`
	Password    string   `json:"password"`
	DisplayName string   `json:"display_name"`
	RoleCodes   []string `json:"role_codes"`
}

type updateAdministratorRequest struct {
	DisplayName *string `json:"display_name"`
}

type resetPasswordRequest struct {
	NewPassword string `json:"new_password"`
}

type assignRolesRequest struct {
	RoleCodes []string `json:"role_codes"`
}

type createRoleRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type updateRoleRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Enabled     *bool   `json:"enabled"`
}

type replacePermissionsRequest struct {
	PermissionCodes []string `json:"permission_codes"`
}

// --- response DTOs ---

type accessTokenData struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type meData struct {
	ID              string   `json:"id"`
	Username        string   `json:"username"`
	DisplayName     string   `json:"display_name"`
	Enabled         bool     `json:"enabled"`
	RoleCodes       []string `json:"role_codes"`
	PermissionCodes []string `json:"permission_codes"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

type administratorData struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"display_name"`
	Enabled     bool     `json:"enabled"`
	RoleCodes   []string `json:"role_codes"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type roleData struct {
	ID              string   `json:"id"`
	Code            string   `json:"code"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	BuiltIn         bool     `json:"built_in"`
	Enabled         bool     `json:"enabled"`
	PermissionCodes []string `json:"permission_codes"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

type permissionData struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type auditEventData struct {
	ID               string         `json:"id"`
	ActorID          *string        `json:"actor_id"`
	ActorUsername    string         `json:"actor_username"`
	ActorDisplayName string         `json:"actor_display_name"`
	ActorRoleCodes   []string       `json:"actor_role_codes"`
	Action           string         `json:"action"`
	ResourceType     string         `json:"resource_type"`
	ResourceID       string         `json:"resource_id"`
	Outcome          string         `json:"outcome"`
	Details          map[string]any `json:"details"`
	RequestID        string         `json:"request_id"`
	SourceIP         string         `json:"source_ip"`
	UserAgent        string         `json:"user_agent"`
	EventAt          string         `json:"event_at"`
	CreatedAt        string         `json:"created_at"`
}

type pageData struct {
	Items    any   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

func idString(id int64) string {
	return strconv.FormatInt(id, 10)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func stringSliceOrEmpty(in []string) []string {
	if in == nil {
		return []string{}
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func toMeData(m *MeResult) meData {
	if m == nil {
		return meData{RoleCodes: []string{}, PermissionCodes: []string{}}
	}
	return meData{
		ID:              idString(m.ID),
		Username:        m.Username,
		DisplayName:     m.DisplayName,
		Enabled:         m.Enabled,
		RoleCodes:       stringSliceOrEmpty(m.RoleCodes),
		PermissionCodes: stringSliceOrEmpty(m.PermissionCodes),
		CreatedAt:       formatTime(m.CreatedAt),
		UpdatedAt:       formatTime(m.UpdatedAt),
	}
}

func toAdministratorData(a *adminauth.Administrator) administratorData {
	if a == nil {
		return administratorData{RoleCodes: []string{}}
	}
	return administratorData{
		ID:          idString(a.ID),
		Username:    a.Username,
		DisplayName: a.DisplayName,
		Enabled:     a.Enabled,
		RoleCodes:   stringSliceOrEmpty(a.RoleCodes),
		CreatedAt:   formatTime(a.CreatedAt),
		UpdatedAt:   formatTime(a.UpdatedAt),
	}
}

func toAdministratorList(items []adminauth.Administrator) []administratorData {
	out := make([]administratorData, 0, len(items))
	for i := range items {
		out = append(out, toAdministratorData(&items[i]))
	}
	return out
}

func toRoleData(r *adminauth.Role) roleData {
	if r == nil {
		return roleData{PermissionCodes: []string{}}
	}
	return roleData{
		ID:              idString(r.ID),
		Code:            r.Code,
		Name:            r.Name,
		Description:     r.Description,
		BuiltIn:         r.BuiltIn,
		Enabled:         r.Enabled,
		PermissionCodes: stringSliceOrEmpty(r.PermissionCodes),
		CreatedAt:       formatTime(r.CreatedAt),
		UpdatedAt:       formatTime(r.UpdatedAt),
	}
}

func toRoleList(items []adminauth.Role) []roleData {
	out := make([]roleData, 0, len(items))
	for i := range items {
		out = append(out, toRoleData(&items[i]))
	}
	return out
}

func toPermissionData(p *adminauth.Permission) permissionData {
	if p == nil {
		return permissionData{}
	}
	return permissionData{
		ID:          idString(p.ID),
		Code:        p.Code,
		Name:        p.Name,
		Description: p.Description,
		CreatedAt:   formatTime(p.CreatedAt),
		UpdatedAt:   formatTime(p.UpdatedAt),
	}
}

func toPermissionList(items []adminauth.Permission) []permissionData {
	out := make([]permissionData, 0, len(items))
	for i := range items {
		out = append(out, toPermissionData(&items[i]))
	}
	return out
}

func toAuditEventData(e *adminauth.AuditEvent) auditEventData {
	if e == nil {
		return auditEventData{
			ActorRoleCodes: []string{},
			Details:        map[string]any{},
		}
	}
	var actorID *string
	if e.ActorID != nil {
		s := idString(*e.ActorID)
		actorID = &s
	}
	details := e.Details
	if details == nil {
		details = map[string]any{}
	}
	return auditEventData{
		ID:               idString(e.ID),
		ActorID:          actorID,
		ActorUsername:    e.ActorUsername,
		ActorDisplayName: e.ActorDisplayName,
		ActorRoleCodes:   stringSliceOrEmpty(e.ActorRoleCodes),
		Action:           e.Action,
		ResourceType:     e.ResourceType,
		ResourceID:       e.ResourceID,
		Outcome:          e.Outcome,
		Details:          details,
		RequestID:        e.RequestID,
		SourceIP:         e.SourceIP,
		UserAgent:        e.UserAgent,
		EventAt:          formatTime(e.EventAt),
		CreatedAt:        formatTime(e.CreatedAt),
	}
}

func toAuditEventList(items []adminauth.AuditEvent) []auditEventData {
	out := make([]auditEventData, 0, len(items))
	for i := range items {
		out = append(out, toAuditEventData(&items[i]))
	}
	return out
}
