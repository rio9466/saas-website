package adminauth

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	"github.com/rio9466/easy-admin/server/internal/platform/auth"
	"github.com/rio9466/easy-admin/server/internal/repository/logdb"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
)

const defaultRefreshCookiePath = "/api/v1/admin/auth"

// AuthOptions carries cookie and origin settings for handlers and service helpers.
type AuthOptions struct {
	TrustedOrigins      []string
	RefreshCookieName   string
	RefreshCookiePath   string
	RefreshCookieSecure bool
	Environment         string
}

// Actor is the authenticated or request-scoped identity used for audits and guards.
type Actor struct {
	ID          int64
	Username    string
	DisplayName string
	RoleCodes   []string
	SessionID   string
	RequestID   string
	SourceIP    string
	UserAgent   string
}

// passwordCrypto is the minimal password primitive the service needs. The
// concrete *auth.PasswordHasher satisfies it; tests substitute a recorder to
// prove dummy comparisons execute.
type passwordCrypto interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

// dummyCredential is hashed once at service construction at the configured
// bcrypt cost and compared against when a login username does not exist, so
// nonexistent and wrong-password logins burn equivalent work.
const dummyCredential = "easy-admin-srv-003-dummy-credential-do-not-use"

// ErrDummyCredentialUnavailable reports that the startup dummy bcrypt hash
// could not be prepared. Construction must fail closed in this state: if the
// service started anyway, unknown-user logins could silently skip the
// configured-cost comparison and let an attacker distinguish existing from
// nonexistent usernames by timing.
var ErrDummyCredentialUnavailable = errors.New("dummy credential hash unavailable")

// Service implements administrator authentication, RBAC, and audit use cases.
type Service struct {
	admins    *primary.AdminRepository
	audits    auditStore
	tokens    *auth.TokenService
	sessions  *auth.SessionStore
	passwords passwordCrypto
	logger    *slog.Logger
	opts      AuthOptions
	dummyHash string
}

type auditStore interface {
	CreatePending(ctx context.Context, event *adminauth.AuditEvent) (int64, error)
	Finalize(ctx context.Context, id int64, resourceID string, outcome string, details map[string]any) error
	List(ctx context.Context, filter logdb.AuditListFilter) (adminauth.Page[adminauth.AuditEvent], error)
	GetByID(ctx context.Context, id int64) (*adminauth.AuditEvent, error)
}

// New constructs a Service. logger may be nil (defaults to slog.Default). The
// configured-cost dummy bcrypt hash used to equalize unknown-user login work is
// derived here and must succeed; a failure (hash error, empty hash, or a nil
// password hasher) returns a sanitized error and no Service so unknown-user
// logins can never silently skip the comparison.
func New(
	admins *primary.AdminRepository,
	audits auditStore,
	tokens *auth.TokenService,
	sessions *auth.SessionStore,
	passwords passwordCrypto,
	logger *slog.Logger,
	opts AuthOptions,
) (*Service, error) {
	if logger == nil {
		logger = slog.Default()
	}
	opts = normalizeAuthOptions(opts)
	if passwords == nil {
		return nil, errors.New("password hasher is required")
	}
	dummy, err := passwords.Hash(dummyCredential)
	if err != nil || dummy == "" {
		// Deliberately not wrapping the raw cause: it is logged nowhere and could
		// carry environment-specific detail, while ErrDummyCredentialUnavailable
		// is enough for a caller to fail startup with a sanitized message.
		return nil, ErrDummyCredentialUnavailable
	}
	return &Service{
		admins:    admins,
		audits:    audits,
		tokens:    tokens,
		sessions:  sessions,
		passwords: passwords,
		logger:    logger,
		opts:      opts,
		dummyHash: dummy,
	}, nil
}

// compareDummyPassword executes one bcrypt comparison against the startup
// dummy hash so a nonexistent username costs the same as a wrong password.
// The comparison result is irrelevant and never influences the response. A
// Service can only be obtained from New, which guarantees dummyHash is set.
func (s *Service) compareDummyPassword(password string) {
	_ = s.passwords.Compare(s.dummyHash, password)
}

// Options returns a copy of the service auth/cookie options for handlers.
func (s *Service) Options() AuthOptions {
	return s.opts
}

func normalizeAuthOptions(opts AuthOptions) AuthOptions {
	if strings.TrimSpace(opts.RefreshCookiePath) == "" {
		opts.RefreshCookiePath = defaultRefreshCookiePath
	}
	return opts
}

// HasPermission reports whether adminID currently holds permission code.
// super_admin bypasses ordinary permission checks.
func (s *Service) HasPermission(ctx context.Context, adminID int64, code string) (bool, error) {
	code = strings.TrimSpace(code)
	if adminID <= 0 || code == "" {
		return false, nil
	}
	roles, err := s.admins.RoleCodesForAdmin(ctx, adminID)
	if err != nil {
		return false, mapError(err)
	}
	if containsRole(roles, adminauth.RoleSuperAdmin) {
		return true, nil
	}
	perms, err := s.admins.EffectivePermissionCodes(ctx, adminID)
	if err != nil {
		return false, mapError(err)
	}
	return containsCode(perms, code), nil
}

func (s *Service) requirePermission(ctx context.Context, actor Actor, code string) error {
	ok, err := s.HasPermission(ctx, actor.ID, code)
	if err != nil {
		return err
	}
	if !ok {
		return apperr.New(403, apperr.CodeForbidden, "forbidden", apperr.ErrForbidden)
	}
	return nil
}

func (s *Service) requireAuth(actor Actor) error {
	if actor.ID <= 0 || strings.TrimSpace(actor.SessionID) == "" {
		return apperr.New(401, apperr.CodeUnauthorized, "unauthorized", apperr.ErrUnauthorized)
	}
	return nil
}

func containsRole(codes []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, code := range codes {
		if strings.ToLower(strings.TrimSpace(code)) == want {
			return true
		}
	}
	return false
}

func containsCode(codes []string, want string) bool {
	return containsRole(codes, want)
}

func actorIsSuper(actor Actor) bool {
	return containsRole(actor.RoleCodes, adminauth.RoleSuperAdmin)
}

func normalizeRoleCodes(codes []string) []string {
	seen := make(map[string]struct{}, len(codes))
	out := make([]string, 0, len(codes))
	for _, code := range codes {
		n := strings.ToLower(strings.TrimSpace(code))
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

func cloneStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

// revokeSessionsBestEffort cleans up Redis session records after an audited
// primary write already bumped the administrator auth epoch. The epoch check is
// the authoritative, race-safe invalidation: it applies even to a session
// created concurrently that the Redis index scan could not observe. A cleanup
// failure only degrades hygiene, never security, and is logged without secrets.
func (s *Service) revokeSessionsBestEffort(ctx context.Context, adminID int64, reason string) {
	if err := s.sessions.RevokeAllForAdmin(ctx, adminID); err != nil {
		s.logger.Error("session revocation cleanup failed; auth epoch invalidates old sessions",
			"administrator_id", idString(adminID),
			"reason", reason,
		)
	}
}
