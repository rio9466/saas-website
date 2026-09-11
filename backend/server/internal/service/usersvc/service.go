package usersvc

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	"github.com/rio9466/easy-admin/server/internal/platform/auth"
	"github.com/rio9466/easy-admin/server/internal/platform/mailer"
	"github.com/rio9466/easy-admin/server/internal/platform/ratelimit"
	"github.com/rio9466/easy-admin/server/internal/platform/secrets"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
)

// DefaultVerificationTokenTTL bounds one-time email verification tokens when
// the configuration does not override it.
const DefaultVerificationTokenTTL = 24 * 60 * 60 * 1_000_000_000 // 24h in nanoseconds

// Password hashing/dummy-credential configuration mirrors the administrator
// surface so unknown-user logins burn equivalent work.
const dummyCredential = "easy-admin-usr-001-dummy-credential-do-not-use"

// ErrDummyCredentialUnavailable reports that the startup dummy bcrypt hash
// could not be prepared; construction must fail closed.
var ErrDummyCredentialUnavailable = errors.New("dummy credential hash unavailable")

// Options carries user-service runtime settings.
type Options struct {
	// VerificationTokenTTL bounds one-time verification tokens; zero uses the
	// documented default of 24 hours.
	VerificationTokenTTL int64 // nanoseconds
	// Environment reflects the deployment environment (development/production).
	Environment string
	// NewMailer builds the SMTP mailer for verification emails. Nil uses
	// mailer.New. Tests inject a fake to avoid external mail.
	NewMailer func(cfg mailer.Settings) (mailer.Mailer, error)
}

// Actor is the request-scoped identity for audits and permission checks.
type Actor struct {
	ID        int64
	Username  string
	Email     string
	Nickname  string
	SessionID string
	RequestID string
	SourceIP  string
	UserAgent string
	// Administrator actor id set only on admin-facing operations.
	AdminID          int64
	AdminUsername    string
	AdminDisplayName string
	AdminRoleCodes   []string
}

// Service implements business-user platform use cases: public settings,
// authentication, verification, levels, points, and administrator management.
type Service struct {
	users     *primary.UserRepository
	admins    *primary.AdminRepository
	audits    auditStore
	tokens    *auth.TokenService
	sessions  *auth.SessionStore
	passwords passwordCrypto
	box       *secrets.Box
	limiter   *ratelimit.Limiter
	logger    *slog.Logger
	opts      Options
	dummyHash string
	verifyTTL int64
	newMailer func(cfg mailer.Settings) (mailer.Mailer, error)
}

// passwordCrypto is the minimal password primitive the service needs.
type passwordCrypto interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

type auditStore interface {
	CreatePending(ctx context.Context, event *adminauth.AuditEvent) (int64, error)
	Finalize(ctx context.Context, id int64, resourceID string, outcome string, details map[string]any) error
}

// New constructs a Service. passwords is required and must produce a dummy
// bcrypt hash; box may be nil when no SMTP master key is configured (email
// verification stays disabled). limiter must be non-nil so rate limiting fails
// closed. logger may be nil (defaults to slog.Default()).
func New(users *primary.UserRepository, admins *primary.AdminRepository, audits auditStore, tokens *auth.TokenService, sessions *auth.SessionStore, passwords passwordCrypto, box *secrets.Box, limiter *ratelimit.Limiter, logger *slog.Logger, opts Options) (*Service, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if passwords == nil {
		return nil, errors.New("password hasher is required")
	}
	if limiter == nil {
		return nil, errors.New("rate limiter is required")
	}
	dummy, err := passwords.Hash(dummyCredential)
	if err != nil || dummy == "" {
		return nil, ErrDummyCredentialUnavailable
	}
	ttl := opts.VerificationTokenTTL
	if ttl <= 0 {
		ttl = DefaultVerificationTokenTTL
	}
	newMailer := opts.NewMailer
	if newMailer == nil {
		newMailer = func(cfg mailer.Settings) (mailer.Mailer, error) {
			return mailer.New(cfg)
		}
	}
	return &Service{
		users:     users,
		admins:    admins,
		audits:    audits,
		tokens:    tokens,
		sessions:  sessions,
		passwords: passwords,
		box:       box,
		limiter:   limiter,
		logger:    logger,
		opts:      opts,
		dummyHash: dummy,
		verifyTTL: ttl,
		newMailer: newMailer,
	}, nil
}

// Options returns a copy of the service options.
func (s *Service) Options() Options {
	return s.opts
}

// compareDummyPassword burns one configured-cost bcrypt comparison so an
// unknown identifier costs the same as a wrong password.
func (s *Service) compareDummyPassword(password string) {
	_ = s.passwords.Compare(s.dummyHash, password)
}

func (s *Service) requireAuth(actor Actor) error {
	if actor.ID <= 0 || strings.TrimSpace(actor.SessionID) == "" {
		return apperr.New(401, apperr.CodeUnauthorized, "unauthorized", apperr.ErrUnauthorized)
	}
	return nil
}

// requireAdminActor rejects request actors that did not authenticate as a
// backend administrator (admin operations run under the administrator surface).
func (s *Service) requireAdminActor(actor Actor) error {
	if actor.AdminID <= 0 {
		return apperr.New(401, apperr.CodeUnauthorized, "unauthorized", apperr.ErrUnauthorized)
	}
	return nil
}

// requirePermission re-checks backend RBAC from primary storage on every
// admin-facing mutation; hiding UI controls is never authorization.
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
	const digits = "0123456789"
	if id == 0 {
		return "0"
	}
	neg := id < 0
	if neg {
		id = -id
	}
	var buf [20]byte
	i := len(buf)
	for id > 0 {
		i--
		buf[i] = digits[id%10]
		id /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func cloneStrings(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func containsCode(codes []string, want string) bool {
	for _, c := range codes {
		if strings.TrimSpace(c) == want {
			return true
		}
	}
	return false
}
