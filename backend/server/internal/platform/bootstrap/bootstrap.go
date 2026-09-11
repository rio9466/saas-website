// Package bootstrap performs the idempotent first-run initialization of the
// management platform. It is part of the single server startup path:
//
//  1. Ensure the built-in super_admin role exists; when it exists its name and
//     description are updated, idempotently, to the approved Chinese values.
//  2. When the database has no administrator at all, create the initial
//     "admin" account with the super_admin role from bootstrap credentials.
//
// It never deletes or overwrites existing administrators, roles, permissions,
// or business data, and it never logs or returns credentials. Every step is
// safe to repeat: repeated startups and concurrent processes converge without
// duplicate rows and without errors.
package bootstrap

import (
	"context"
	"errors"
	"strings"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	platformauth "github.com/rio9466/easy-admin/server/internal/platform/auth"
	"github.com/rio9466/easy-admin/server/internal/platform/postgres"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
	"golang.org/x/crypto/bcrypt"
)

// Environment variable names for first-run bootstrap credentials.
const (
	EnvBootstrapUsername = "BOOTSTRAP_ADMIN_USERNAME"
	EnvBootstrapPassword = "BOOTSTRAP_ADMIN_PASSWORD"
	EnvBootstrapDisplay  = "BOOTSTRAP_ADMIN_DISPLAY_NAME"
)

// requiredInitialUsername is the fixed username of the initial administrator
// account created on a fresh database. BOOTSTRAP_ADMIN_USERNAME may only ever
// restate this value; any other username aborts startup (work order
// admin-platform-startup-20260907: fresh environments get exactly one `admin`
// account, never root/operator/...).
const requiredInitialUsername = defaultInitialUsername

const (
	// defaultInitialUsername is the only administrator account created on a
	// fresh database.
	defaultInitialUsername = "admin"
	// devDefaultPassword is the documented local development fallback; it is
	// only applied when auth.environment = development and neither
	// BOOTSTRAP_ADMIN_USERNAME nor BOOTSTRAP_ADMIN_PASSWORD is configured.
	// Never logged, never accepted in production.
	devDefaultPassword = "admin123"
	// defaultDisplayName is used when BOOTSTRAP_ADMIN_DISPLAY_NAME is unset.
	defaultDisplayName = "超级管理员"

	// Approved Chinese name/description for the built-in super_admin role.
	superAdminRoleName        = "超级管理员"
	superAdminRoleDescription = "超级管理员"
)

// Options carries resolved bootstrap credentials and hashing settings.
type Options struct {
	Username    string
	Password    string
	DisplayName string
	// Environment is the normalized auth.environment value.
	Environment string
	// BcryptCost is the configured hashing cost.
	BcryptCost int
	// usingDevDefaults records that the development fallback credentials are
	// in effect (they bypass the ordinary length validation, like the
	// previous bootstrap-admin command did).
	usingDevDefaults bool
}

// Result summarizes one Ensure call.
type Result struct {
	// RoleEnsured reports that the super_admin role existed or was created.
	RoleEnsured bool
	// RoleUpdated reports that the role name/description changed this run.
	RoleUpdated bool
	// AdminCreated reports that the initial administrator was created.
	AdminCreated bool
	// AdminExists reports that administrators were already present.
	AdminExists bool
}

// CredentialsFromEnv resolves bootstrap credentials from the environment via
// lookup. In development, when nothing is configured, the documented local
// defaults apply so a fresh `make run` works without extra setup. Production
// never applies defaults and rejects them when explicitly configured.
func CredentialsFromEnv(lookup func(string) (string, bool), environment string) (Options, error) {
	env := normalizeEnv(environment)
	configuredUsername := strings.TrimSpace(envValue(lookup, EnvBootstrapUsername))
	password := envValue(lookup, EnvBootstrapPassword)
	displayName := strings.TrimSpace(envValue(lookup, EnvBootstrapDisplay))
	if displayName == "" {
		displayName = defaultDisplayName
	}

	// The initial account is always `admin`. A configured username may only
	// restate it; anything else is a configuration error and must abort.
	username := configuredUsername
	if username == "" {
		username = requiredInitialUsername
	}
	if err := validateBootstrapUsername(lookup); err != nil {
		return Options{}, err
	}

	if username == requiredInitialUsername && password == "" && env == "development" {
		password = devDefaultPassword
	}
	if password == "" {
		return Options{}, errors.New("bootstrap credentials required via BOOTSTRAP_ADMIN_PASSWORD (username is fixed to " + requiredInitialUsername + ")")
	}

	opts := Options{
		Username:    username,
		Password:    password,
		DisplayName: displayName,
		Environment: env,
	}

	if env == "production" && isDevDefaults(opts) {
		return Options{}, errors.New("production bootstrap rejects development default credentials")
	}
	opts.usingDevDefaults = env == "development" && isDevDefaults(opts)
	if !opts.usingDevDefaults {
		if err := platformauth.ValidatePasswordLength(password); err != nil {
			return Options{}, errors.New("invalid bootstrap password length")
		}
	}
	return opts, nil
}

// Ensure runs the idempotent bootstrap sequence against the primary database
// and reports what changed. When administrators already exist, no credentials
// are required and no account is created or modified. A misconfigured
// BOOTSTRAP_ADMIN_USERNAME (anything other than the fixed `admin` account)
// fails startup regardless of database state.
func Ensure(ctx context.Context, db *postgres.DB, lookup func(string) (string, bool), environment string, bcryptCost int) (Result, error) {
	if db == nil {
		return Result{}, errors.New("bootstrap: primary database is not initialized")
	}
	if err := validateBootstrapUsername(lookup); err != nil {
		return Result{}, err
	}
	repo := primary.NewAdminRepository(db.GORM())

	result, err := ensureSuperAdminRole(ctx, repo)
	if err != nil {
		return result, err
	}

	count, err := repo.CountAdministrators(ctx)
	if err != nil {
		return result, errors.New("bootstrap: count administrators failed")
	}
	if count > 0 {
		result.AdminExists = true
		return result, nil
	}

	opts, err := CredentialsFromEnv(lookup, environment)
	if err != nil {
		return result, err
	}
	opts.BcryptCost = bcryptCost

	role, err := repo.GetRoleByCode(ctx, adminauth.RoleSuperAdmin)
	if err != nil {
		return result, errors.New("bootstrap: load super_admin role failed")
	}

	hash, err := hashPassword(opts)
	if err != nil {
		return result, err
	}

	admin := &adminauth.Administrator{
		Username:     opts.Username,
		PasswordHash: hash,
		DisplayName:  opts.DisplayName,
		Enabled:      true,
	}
	if err := repo.BootstrapSuperAdmin(ctx, admin, role.ID); err != nil {
		if errors.Is(err, primary.ErrBootstrapAlreadyCompleted) {
			// Another process finished bootstrap first: keep the existing
			// account untouched and treat this as success.
			result.AdminExists = true
			return result, nil
		}
		return result, errors.New("bootstrap: create initial administrator failed")
	}
	result.AdminCreated = true
	return result, nil
}

// ensureSuperAdminRole guarantees the built-in super_admin role exists with
// the approved Chinese name/description. Updates are name/description only
// and skipped when already correct, so repeated startups are no-ops.
func ensureSuperAdminRole(ctx context.Context, repo *primary.AdminRepository) (Result, error) {
	result := Result{RoleEnsured: true}
	role, err := repo.GetRoleByCode(ctx, adminauth.RoleSuperAdmin)
	if err != nil {
		if !errors.Is(err, primary.ErrNotFound) {
			return result, errors.New("bootstrap: load super_admin role failed")
		}
		builtIn := true
		enabled := true
		if err := repo.CreateRole(ctx, &adminauth.Role{
			Code:        adminauth.RoleSuperAdmin,
			Name:        superAdminRoleName,
			Description: superAdminRoleDescription,
			BuiltIn:     builtIn,
			Enabled:     enabled,
		}); err != nil {
			return result, errors.New("bootstrap: create super_admin role failed")
		}
		return result, nil
	}

	if role.Name == superAdminRoleName && role.Description == superAdminRoleDescription {
		return result, nil
	}
	name := superAdminRoleName
	description := superAdminRoleDescription
	if err := repo.UpdateRole(ctx, role.ID, &name, &description, nil); err != nil {
		return result, errors.New("bootstrap: update super_admin role name failed")
	}
	result.RoleUpdated = true
	return result, nil
}

func hashPassword(opts Options) (string, error) {
	if opts.usingDevDefaults {
		if opts.BcryptCost < platformauth.MinBcryptCost {
			return "", errors.New("bootstrap: bcrypt cost too low")
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(opts.Password), opts.BcryptCost)
		if err != nil {
			return "", errors.New("bootstrap: hash password failed")
		}
		return string(hashed), nil
	}
	hasher, err := platformauth.NewPasswordHasher(opts.BcryptCost)
	if err != nil {
		return "", errors.New("bootstrap: configure password hasher failed")
	}
	hash, err := hasher.Hash(opts.Password)
	if err != nil {
		return "", errors.New("bootstrap: hash password failed")
	}
	return hash, nil
}

func normalizeEnv(environment string) string {
	return strings.ToLower(strings.TrimSpace(environment))
}

func envValue(lookup func(string) (string, bool), key string) string {
	if lookup == nil {
		return ""
	}
	value, _ := lookup(key)
	return value
}

func isDevDefaults(opts Options) bool {
	return opts.Username == requiredInitialUsername && opts.Password == devDefaultPassword
}

// validateBootstrapUsername fails when BOOTSTRAP_ADMIN_USERNAME names anything
// other than the fixed initial `admin` account. It is checked before any
// database work so a misconfiguration never starts the server.
func validateBootstrapUsername(lookup func(string) (string, bool)) error {
	configured := strings.TrimSpace(envValue(lookup, EnvBootstrapUsername))
	if configured == "" || configured == requiredInitialUsername {
		return nil
	}
	return errors.New("bootstrap administrator username is fixed to " + requiredInitialUsername +
		"; BOOTSTRAP_ADMIN_USERNAME must be unset or " + requiredInitialUsername)
}
