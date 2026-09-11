// Command server is the single startup entry point of easy-admin. One process
// performs, in order and fail-closed:
//
//  1. configuration loading (TOML file, then named environment overrides),
//  2. primary PostgreSQL connection check,
//  3. primary database migrations (explicit SQL, idempotent),
//  4. log PostgreSQL connection check,
//  5. log database migrations (explicit SQL, idempotent),
//  6. idempotent bootstrap of the initial administrator and super_admin role,
//  7. dependency wiring (Redis, JWT, session stores, services),
//  8. HTTP listener startup with graceful shutdown.
//
// Any failure before step 8 aborts startup with a non-zero exit code; the
// process never listens on the HTTP port in that case. The separate
// cmd/migrate and cmd/bootstrap-admin entry points were removed: the startup
// path below owns migration and bootstrap execution.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rio9466/easy-admin/server/internal/config"
	platformauth "github.com/rio9466/easy-admin/server/internal/platform/auth"
	"github.com/rio9466/easy-admin/server/internal/platform/bootstrap"
	"github.com/rio9466/easy-admin/server/internal/platform/migrate"
	"github.com/rio9466/easy-admin/server/internal/platform/postgres"
	"github.com/rio9466/easy-admin/server/internal/platform/ratelimit"
	appredis "github.com/rio9466/easy-admin/server/internal/platform/redis"
	"github.com/rio9466/easy-admin/server/internal/platform/secrets"
	"github.com/rio9466/easy-admin/server/internal/repository/logdb"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
	adminsvc "github.com/rio9466/easy-admin/server/internal/service/adminauth"
	usersvc "github.com/rio9466/easy-admin/server/internal/service/usersvc"
	transporthttp "github.com/rio9466/easy-admin/server/internal/transport/http"
	"github.com/rio9466/easy-admin/server/internal/transport/http/handler"
)

const (
	msgConnectPrimaryPostgresFailed = "connect primary postgres failed"
	msgConnectLogPostgresFailed     = "connect log postgres failed"
	msgConnectRedisFailed           = "connect redis failed"
	msgClosePrimaryPostgresFailed   = "close primary postgres failed"
	msgCloseLogPostgresFailed       = "close log postgres failed"
	msgCloseRedisFailed             = "close redis failed"

	// migrationRoot is resolved relative to the working directory; the server
	// must be started from the server/ directory (make run does this).
	migrationRoot = "migrations"

	// migrationTimeout bounds one migration target per startup.
	migrationTimeout = 2 * time.Minute
)

func main() {
	configPath := flag.String("config", "configs/config.local.toml", "path to TOML configuration file")
	flag.Parse()

	if err := run(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

func run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := newLogger(cfg.Log.Level)
	logger.Info("starting server",
		"addr", cfg.Server.Addr,
		"log_level", strings.ToLower(cfg.Log.Level),
		"environment", strings.ToLower(cfg.Auth.Environment),
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Primary PostgreSQL: connectivity check, then explicit SQL migrations.
	primaryDB, err := openPostgres(ctx, cfg.Database.Primary, msgConnectPrimaryPostgresFailed)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := closePostgres(primaryDB, msgClosePrimaryPostgresFailed); closeErr != nil {
			logger.Error(msgClosePrimaryPostgresFailed)
		}
	}()

	primaryResult, err := applyMigrations(ctx, cfg.Database.Primary.DSN, "primary")
	if err != nil {
		return err
	}
	logger.Info("primary migrations current",
		"applied", len(primaryResult.Applied),
		"skipped", primaryResult.Skipped,
	)

	// Log PostgreSQL: connectivity check, then explicit SQL migrations.
	logDB, err := openPostgres(ctx, cfg.Database.Log, msgConnectLogPostgresFailed)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := closePostgres(logDB, msgCloseLogPostgresFailed); closeErr != nil {
			logger.Error(msgCloseLogPostgresFailed)
		}
	}()

	logResult, err := applyMigrations(ctx, cfg.Database.Log.DSN, "log")
	if err != nil {
		return err
	}
	logger.Info("log migrations current",
		"applied", len(logResult.Applied),
		"skipped", logResult.Skipped,
	)

	// Idempotent bootstrap: guarantees the super_admin role (Chinese name and
	// description) and creates the initial administrator on a fresh database
	// only. Existing administrators, roles, permissions, and business data are
	// never deleted or overwritten.
	bootstrapResult, err := bootstrap.Ensure(ctx, primaryDB, os.LookupEnv, cfg.Auth.Environment, cfg.Auth.BcryptCost)
	if err != nil {
		return err
	}
	logger.Info("bootstrap ensured",
		"admin_created", bootstrapResult.AdminCreated,
		"admin_exists", bootstrapResult.AdminExists,
		"role_updated", bootstrapResult.RoleUpdated,
	)

	redisClient, err := openRedis(ctx, cfg.Redis)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := closeRedis(redisClient); closeErr != nil {
			logger.Error(msgCloseRedisFailed)
		}
	}()

	tokens, err := platformauth.NewTokenService(platformauth.JWTConfig{
		Secret:   cfg.Auth.JWTSecret,
		Issuer:   cfg.Auth.JWTIssuer,
		Audience: cfg.Auth.JWTAudience,
	})
	if err != nil {
		return errors.New("configure jwt failed")
	}
	sessions, err := platformauth.NewSessionStore(redisClient.Raw())
	if err != nil {
		return errors.New("configure session store failed")
	}
	passwords, err := platformauth.NewPasswordHasher(cfg.Auth.BcryptCost)
	if err != nil {
		return errors.New("configure password hasher failed")
	}

	adminRepo := primary.NewAdminRepository(primaryDB.GORM())
	auditRepo := logdb.NewAuditRepository(logDB.GORM())
	svc, err := adminsvc.New(adminRepo, auditRepo, tokens, sessions, passwords, logger, adminsvc.AuthOptions{
		TrustedOrigins:      append([]string(nil), cfg.Auth.TrustedOrigins...),
		RefreshCookieName:   cfg.Auth.RefreshCookieName,
		RefreshCookiePath:   cfg.Auth.RefreshCookiePath,
		RefreshCookieSecure: cfg.Auth.RefreshCookieSecure,
		Environment:         cfg.Auth.Environment,
	})
	if err != nil {
		return errors.New("configure admin auth service failed")
	}
	adapter := &handler.ServiceAdapter{Svc: svc}

	// Business-user platform wiring: distinct JWT audience, refresh cookie,
	// and Redis session namespace; separate SMTP secret box and rate limiter.
	userTokens, err := platformauth.NewTokenService(platformauth.JWTConfig{
		Secret:   cfg.Auth.JWTSecret,
		Issuer:   cfg.Auth.JWTIssuer,
		Audience: cfg.Auth.UserJWTAudience,
	})
	if err != nil {
		return errors.New("configure user jwt failed")
	}
	userSessions, err := platformauth.NewUserSessionStore(redisClient.Raw())
	if err != nil {
		return errors.New("configure user session store failed")
	}
	limiter, err := ratelimit.New(redisClient.Raw())
	if err != nil {
		return errors.New("configure rate limiter failed")
	}
	var secretBox *secrets.Box
	if cfg.User.SMTPMasterKey != "" {
		secretBox, err = secrets.NewBox([]byte(cfg.User.SMTPMasterKey))
		if err != nil {
			return errors.New("configure smtp secret box failed")
		}
	}
	userRepo := primary.NewUserRepository(primaryDB.GORM())
	userSvc, err := usersvc.New(userRepo, adminRepo, auditRepo, userTokens, userSessions, passwords, secretBox, limiter, logger, usersvc.Options{
		VerificationTokenTTL: int64(cfg.User.VerificationTokenTTL),
		Environment:          cfg.Auth.Environment,
	})
	if err != nil {
		return errors.New("configure business user service failed")
	}
	userAdapter := &handler.UserServiceAdapter{Svc: userSvc}

	ready := handler.MultiReady(logger,
		handler.NamedReadyCheck{Name: "primary_postgres", Checker: primaryDB, Timeout: cfg.Database.Primary.PingTimeout},
		handler.NamedReadyCheck{Name: "log_postgres", Checker: logDB, Timeout: cfg.Database.Log.PingTimeout},
		handler.NamedReadyCheck{Name: "redis", Checker: redisClient, Timeout: cfg.Redis.PingTimeout},
	)

	router := transporthttp.NewRouter(transporthttp.Dependencies{
		Logger:       logger,
		ReadyChecker: ready,
		AdminAuth:    adapter,
		UserClient:   userAdapter,
		UserAdmin:    userAdapter,
		Tokens:       tokens,
		Sessions:     sessions,
		UserTokens:   userTokens,
		UserSessions: userSessions,
		Cookie: handler.AuthCookieSettings{
			Name:   cfg.Auth.RefreshCookieName,
			Path:   cfg.Auth.RefreshCookiePath,
			Secure: cfg.Auth.RefreshCookieSecure,
		},
		UserCookie: handler.UserCookieSettings{
			Name:   cfg.Auth.UserRefreshCookieName,
			Path:   cfg.Auth.UserRefreshCookiePath,
			Secure: cfg.Auth.UserRefreshCookieSecure || cfg.Auth.RefreshCookieSecure,
		},
		TrustedOrigins: append([]string(nil), cfg.Auth.TrustedOrigins...),
	})

	srv := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           router,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("http server listening", "addr", cfg.Server.Addr)
		if serveErr := srv.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case serveErr := <-errCh:
		if serveErr != nil {
			return fmt.Errorf("http server: %w", serveErr)
		}
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	if serveErr := <-errCh; serveErr != nil {
		return fmt.Errorf("http server: %w", serveErr)
	}
	logger.Info("server stopped")
	return nil
}

// applyMigrations applies all pending "up" migrations for the target database.
// It returns a fixed safe message on failure so DSNs or driver internals never
// reach logs; the migration version is preserved for diagnosability.
func applyMigrations(ctx context.Context, dsn, target string) (migrate.Result, error) {
	migrateCtx, cancel := context.WithTimeout(ctx, migrationTimeout)
	defer cancel()

	result, err := migrate.Run(migrateCtx, dsn, migrationRoot, target)
	if err != nil {
		return migrate.Result{}, fmt.Errorf("%s migrations: %w", target, err)
	}
	return result, nil
}

func openPostgres(ctx context.Context, cfg config.PostgresConfig, safeMsg string) (*postgres.DB, error) {
	db, err := postgres.Open(ctx, cfg)
	if err != nil {
		return nil, errors.New(safeMsg)
	}
	return db, nil
}

func closePostgres(db *postgres.DB, safeMsg string) error {
	if err := db.Close(); err != nil {
		return errors.New(safeMsg)
	}
	return nil
}

func openRedis(ctx context.Context, cfg config.RedisConfig) (*appredis.Client, error) {
	client, err := appredis.Open(ctx, cfg)
	if err != nil {
		return nil, errors.New(msgConnectRedisFailed)
	}
	return client, nil
}

func closeRedis(client *appredis.Client) error {
	if err := client.Close(); err != nil {
		return errors.New(msgCloseRedisFailed)
	}
	return nil
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}
