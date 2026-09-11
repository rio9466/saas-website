package contentsvc

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
)

// Actor is the request-scoped administrator identity for content operations.
type Actor struct {
	AdminID          int64
	AdminUsername    string
	AdminDisplayName string
	AdminRoleCodes   []string
	RequestID        string
	SourceIP         string
	UserAgent        string
}

// Service implements public content reads and administrator content CRUD with
// RBAC and pending-first audit.
type Service struct {
	content *primary.ContentRepository
	admins  *primary.AdminRepository
	audits  auditStore
	logger  *slog.Logger
}

type auditStore interface {
	CreatePending(ctx context.Context, event *adminauth.AuditEvent) (int64, error)
	Finalize(ctx context.Context, id int64, resourceID string, outcome string, details map[string]any) error
}

// New constructs a content service. All collaborators are required.
func New(contentRepo *primary.ContentRepository, admins *primary.AdminRepository, audits auditStore, logger *slog.Logger) (*Service, error) {
	if contentRepo == nil {
		return nil, errors.New("content repository is required")
	}
	if admins == nil {
		return nil, errors.New("admin repository is required")
	}
	if audits == nil {
		return nil, errors.New("audit store is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{content: contentRepo, admins: admins, audits: audits, logger: logger}, nil
}

// requireAdminActor rejects actors that did not authenticate as administrators.
func (s *Service) requireAdminActor(actor Actor) error {
	if actor.AdminID <= 0 {
		return apperr.New(401, apperr.CodeUnauthorized, "unauthorized", apperr.ErrUnauthorized)
	}
	return nil
}

// requirePermission re-checks RBAC from primary storage on every mutation;
// hiding UI controls is never authorization.
func (s *Service) requirePermission(ctx context.Context, actor Actor, code string) error {
	if err := s.requireAdminActor(actor); err != nil {
		return err
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return apperr.New(403, apperr.CodeForbidden, "forbidden", apperr.ErrForbidden)
	}
	hasSuper, err := s.admins.AdministratorHasRole(ctx, actor.AdminID, adminauth.RoleSuperAdmin)
	if err != nil {
		return mapError(err)
	}
	if hasSuper {
		return nil
	}
	perms, err := s.admins.EffectivePermissionCodes(ctx, actor.AdminID)
	if err != nil {
		return mapError(err)
	}
	for _, p := range perms {
		if p == code {
			return nil
		}
	}
	return apperr.New(403, apperr.CodeForbidden, "forbidden", apperr.ErrForbidden)
}

func idString(id int64) string {
	if id == 0 {
		return "0"
	}
	negative := id < 0
	if negative {
		id = -id
	}
	var buf [20]byte
	i := len(buf)
	for id > 0 {
		i--
		buf[i] = byte('0' + id%10)
		id /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func derefBool(p *bool) bool {
	return p != nil && *p
}
