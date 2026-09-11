package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// Named environment overrides applied after TOML loading.
const (
	EnvServerAddr         = "SERVER_ADDR"
	EnvPrimaryPostgresDSN = "PRIMARY_POSTGRES_DSN"
	EnvLogPostgresDSN     = "LOG_POSTGRES_DSN"
	EnvRedisAddr          = "REDIS_ADDR"
	EnvRedisPassword      = "REDIS_PASSWORD"
	EnvRedisDB            = "REDIS_DB"
	EnvLogLevel           = "LOG_LEVEL"
	EnvAppEnvironment     = "APP_ENVIRONMENT"
	EnvJWTSecret          = "JWT_SECRET"
	EnvJWTIssuer          = "JWT_ISSUER"
	EnvJWTAudience        = "JWT_AUDIENCE"
	// SMTPMasterKey is the 32-byte key protecting stored SMTP passwords.
	EnvSMTPMasterKey = "SMTP_MASTER_KEY"
)

// Config is the validated application configuration loaded once at startup.
type Config struct {
	Server   ServerConfig   `toml:"server"`
	Database DatabaseConfig `toml:"database"`
	Redis    RedisConfig    `toml:"redis"`
	Auth     AuthConfig     `toml:"auth"`
	User     UserConfig     `toml:"user"`
	Log      LogConfig      `toml:"log"`
}

type ServerConfig struct {
	Addr              string        `toml:"addr"`
	ReadHeaderTimeout time.Duration `toml:"read_header_timeout"`
	ReadTimeout       time.Duration `toml:"read_timeout"`
	WriteTimeout      time.Duration `toml:"write_timeout"`
	IdleTimeout       time.Duration `toml:"idle_timeout"`
	ShutdownTimeout   time.Duration `toml:"shutdown_timeout"`
}

type DatabaseConfig struct {
	Primary PostgresConfig `toml:"primary"`
	Log     PostgresConfig `toml:"log"`
}

type PostgresConfig struct {
	DSN             string        `toml:"dsn"`
	MaxOpenConns    int           `toml:"max_open_conns"`
	MaxIdleConns    int           `toml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `toml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `toml:"conn_max_idle_time"`
	StartupTimeout  time.Duration `toml:"startup_timeout"`
	PingTimeout     time.Duration `toml:"ping_timeout"`
}

type RedisConfig struct {
	Addr           string        `toml:"addr"`
	Password       string        `toml:"password"`
	DB             int           `toml:"db"`
	PoolSize       int           `toml:"pool_size"`
	DialTimeout    time.Duration `toml:"dial_timeout"`
	ReadTimeout    time.Duration `toml:"read_timeout"`
	WriteTimeout   time.Duration `toml:"write_timeout"`
	StartupTimeout time.Duration `toml:"startup_timeout"`
	PingTimeout    time.Duration `toml:"ping_timeout"`
}

// AuthConfig holds JWT, cookie, origin, and password settings.
type AuthConfig struct {
	Environment         string   `toml:"environment"`
	JWTSecret           string   `toml:"jwt_secret"`
	JWTIssuer           string   `toml:"jwt_issuer"`
	JWTAudience         string   `toml:"jwt_audience"`
	RefreshCookieName   string   `toml:"refresh_cookie_name"`
	RefreshCookiePath   string   `toml:"refresh_cookie_path"`
	RefreshCookieSecure bool     `toml:"refresh_cookie_secure"`
	TrustedOrigins      []string `toml:"trusted_origins"`
	BcryptCost          int      `toml:"bcrypt_cost"`
	// Business-user authentication is fully separate: distinct audience and
	// refresh cookie identity so user and administrator tokens/cookies can
	// never be confused.
	UserJWTAudience         string `toml:"user_jwt_audience"`
	UserRefreshCookieName   string `toml:"user_refresh_cookie_name"`
	UserRefreshCookiePath   string `toml:"user_refresh_cookie_path"`
	UserRefreshCookieSecure bool   `toml:"user_refresh_cookie_secure"`
}

// UserConfig holds business-user platform settings that are not stored in the
// typed system_settings singleton (secrets, TTLs).
type UserConfig struct {
	// SMTPMasterKey is the 32-byte key that decrypts stored SMTP passwords.
	// Load from SMTP_MASTER_KEY; never from PostgreSQL or Git.
	SMTPMasterKey string `toml:"smtp_master_key"`
	// VerificationTokenTTL bounds one-time email verification tokens.
	VerificationTokenTTL time.Duration `toml:"verification_token_ttl"`
}

type LogConfig struct {
	Level string `toml:"level"`
}

type fileConfig struct {
	Server   fileServerConfig   `toml:"server"`
	Database fileDatabaseConfig `toml:"database"`
	Redis    fileRedisConfig    `toml:"redis"`
	Auth     fileAuthConfig     `toml:"auth"`
	User     fileUserConfig     `toml:"user"`
	Log      LogConfig          `toml:"log"`
}

type fileServerConfig struct {
	Addr              string `toml:"addr"`
	ReadHeaderTimeout string `toml:"read_header_timeout"`
	ReadTimeout       string `toml:"read_timeout"`
	WriteTimeout      string `toml:"write_timeout"`
	IdleTimeout       string `toml:"idle_timeout"`
	ShutdownTimeout   string `toml:"shutdown_timeout"`
}

type fileDatabaseConfig struct {
	Primary filePostgresConfig `toml:"primary"`
	Log     filePostgresConfig `toml:"log"`
}

type filePostgresConfig struct {
	DSN             string `toml:"dsn"`
	MaxOpenConns    int    `toml:"max_open_conns"`
	MaxIdleConns    int    `toml:"max_idle_conns"`
	ConnMaxLifetime string `toml:"conn_max_lifetime"`
	ConnMaxIdleTime string `toml:"conn_max_idle_time"`
	StartupTimeout  string `toml:"startup_timeout"`
	PingTimeout     string `toml:"ping_timeout"`
}

type fileRedisConfig struct {
	Addr           string `toml:"addr"`
	Password       string `toml:"password"`
	DB             int    `toml:"db"`
	PoolSize       int    `toml:"pool_size"`
	DialTimeout    string `toml:"dial_timeout"`
	ReadTimeout    string `toml:"read_timeout"`
	WriteTimeout   string `toml:"write_timeout"`
	StartupTimeout string `toml:"startup_timeout"`
	PingTimeout    string `toml:"ping_timeout"`
}

type fileAuthConfig struct {
	Environment             string   `toml:"environment"`
	JWTSecret               string   `toml:"jwt_secret"`
	JWTIssuer               string   `toml:"jwt_issuer"`
	JWTAudience             string   `toml:"jwt_audience"`
	RefreshCookieName       string   `toml:"refresh_cookie_name"`
	RefreshCookiePath       string   `toml:"refresh_cookie_path"`
	RefreshCookieSecure     bool     `toml:"refresh_cookie_secure"`
	TrustedOrigins          []string `toml:"trusted_origins"`
	BcryptCost              int      `toml:"bcrypt_cost"`
	UserJWTAudience         string   `toml:"user_jwt_audience"`
	UserRefreshCookieName   string   `toml:"user_refresh_cookie_name"`
	UserRefreshCookiePath   string   `toml:"user_refresh_cookie_path"`
	UserRefreshCookieSecure bool     `toml:"user_refresh_cookie_secure"`
}

type fileUserConfig struct {
	SMTPMasterKey        string `toml:"smtp_master_key"`
	VerificationTokenTTL string `toml:"verification_token_ttl"`
}

// Load reads a TOML file, applies named environment overrides, and validates.
// Precedence is deterministic: TOML values first, then environment overrides.
func Load(path string) (Config, error) {
	return load(path, os.LookupEnv)
}

func load(path string, lookup func(string) (string, bool)) (Config, error) {
	if strings.TrimSpace(path) == "" {
		return Config{}, errors.New("config path is required")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}

	var raw fileConfig
	if err := toml.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("parse config %q: %w", path, err)
	}

	cfg, err := raw.toConfig()
	if err != nil {
		return Config{}, err
	}

	if err := applyEnvOverrides(&cfg, lookup); err != nil {
		return Config{}, err
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (raw fileConfig) toConfig() (Config, error) {
	var cfg Config
	var err error

	cfg.Server.Addr = raw.Server.Addr
	if cfg.Server.ReadHeaderTimeout, err = parseDuration("server.read_header_timeout", raw.Server.ReadHeaderTimeout); err != nil {
		return Config{}, err
	}
	if cfg.Server.ReadTimeout, err = parseDuration("server.read_timeout", raw.Server.ReadTimeout); err != nil {
		return Config{}, err
	}
	if cfg.Server.WriteTimeout, err = parseDuration("server.write_timeout", raw.Server.WriteTimeout); err != nil {
		return Config{}, err
	}
	if cfg.Server.IdleTimeout, err = parseDuration("server.idle_timeout", raw.Server.IdleTimeout); err != nil {
		return Config{}, err
	}
	if cfg.Server.ShutdownTimeout, err = parseDuration("server.shutdown_timeout", raw.Server.ShutdownTimeout); err != nil {
		return Config{}, err
	}

	if cfg.Database.Primary, err = raw.Database.Primary.toPostgres("database.primary"); err != nil {
		return Config{}, err
	}
	if cfg.Database.Log, err = raw.Database.Log.toPostgres("database.log"); err != nil {
		return Config{}, err
	}

	cfg.Redis.Addr = raw.Redis.Addr
	cfg.Redis.Password = raw.Redis.Password
	cfg.Redis.DB = raw.Redis.DB
	cfg.Redis.PoolSize = raw.Redis.PoolSize
	if cfg.Redis.DialTimeout, err = parseDuration("redis.dial_timeout", raw.Redis.DialTimeout); err != nil {
		return Config{}, err
	}
	if cfg.Redis.ReadTimeout, err = parseDuration("redis.read_timeout", raw.Redis.ReadTimeout); err != nil {
		return Config{}, err
	}
	if cfg.Redis.WriteTimeout, err = parseDuration("redis.write_timeout", raw.Redis.WriteTimeout); err != nil {
		return Config{}, err
	}
	if cfg.Redis.StartupTimeout, err = parseDuration("redis.startup_timeout", raw.Redis.StartupTimeout); err != nil {
		return Config{}, err
	}
	if cfg.Redis.PingTimeout, err = parseDuration("redis.ping_timeout", raw.Redis.PingTimeout); err != nil {
		return Config{}, err
	}

	cfg.Auth = AuthConfig{
		Environment:             raw.Auth.Environment,
		JWTSecret:               raw.Auth.JWTSecret,
		JWTIssuer:               raw.Auth.JWTIssuer,
		JWTAudience:             raw.Auth.JWTAudience,
		RefreshCookieName:       raw.Auth.RefreshCookieName,
		RefreshCookiePath:       raw.Auth.RefreshCookiePath,
		RefreshCookieSecure:     raw.Auth.RefreshCookieSecure,
		TrustedOrigins:          append([]string(nil), raw.Auth.TrustedOrigins...),
		BcryptCost:              raw.Auth.BcryptCost,
		UserJWTAudience:         raw.Auth.UserJWTAudience,
		UserRefreshCookieName:   raw.Auth.UserRefreshCookieName,
		UserRefreshCookiePath:   raw.Auth.UserRefreshCookiePath,
		UserRefreshCookieSecure: raw.Auth.UserRefreshCookieSecure,
	}
	if strings.TrimSpace(cfg.Auth.UserJWTAudience) == "" {
		cfg.Auth.UserJWTAudience = "easy-admin-user"
	}

	cfg.User.SMTPMasterKey = raw.User.SMTPMasterKey
	ttl := raw.User.VerificationTokenTTL
	if strings.TrimSpace(ttl) == "" {
		ttl = "24h"
	}
	if cfg.User.VerificationTokenTTL, err = parseDuration("user.verification_token_ttl", ttl); err != nil {
		return Config{}, err
	}

	cfg.Log = raw.Log
	return cfg, nil
}

func (raw filePostgresConfig) toPostgres(prefix string) (PostgresConfig, error) {
	var cfg PostgresConfig
	var err error
	cfg.DSN = raw.DSN
	cfg.MaxOpenConns = raw.MaxOpenConns
	cfg.MaxIdleConns = raw.MaxIdleConns
	if cfg.ConnMaxLifetime, err = parseDuration(prefix+".conn_max_lifetime", raw.ConnMaxLifetime); err != nil {
		return PostgresConfig{}, err
	}
	if cfg.ConnMaxIdleTime, err = parseDuration(prefix+".conn_max_idle_time", raw.ConnMaxIdleTime); err != nil {
		return PostgresConfig{}, err
	}
	if cfg.StartupTimeout, err = parseDuration(prefix+".startup_timeout", raw.StartupTimeout); err != nil {
		return PostgresConfig{}, err
	}
	if cfg.PingTimeout, err = parseDuration(prefix+".ping_timeout", raw.PingTimeout); err != nil {
		return PostgresConfig{}, err
	}
	return cfg, nil
}

func parseDuration(field, value string) (time.Duration, error) {
	if strings.TrimSpace(value) == "" {
		return 0, fmt.Errorf("%s is required", field)
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid duration %q", field, value)
	}
	return d, nil
}

func applyEnvOverrides(cfg *Config, lookup func(string) (string, bool)) error {
	if v, ok := lookup(EnvServerAddr); ok && strings.TrimSpace(v) != "" {
		cfg.Server.Addr = v
	}
	if v, ok := lookup(EnvPrimaryPostgresDSN); ok && strings.TrimSpace(v) != "" {
		cfg.Database.Primary.DSN = v
	}
	if v, ok := lookup(EnvLogPostgresDSN); ok && strings.TrimSpace(v) != "" {
		cfg.Database.Log.DSN = v
	}
	if v, ok := lookup(EnvRedisAddr); ok && strings.TrimSpace(v) != "" {
		cfg.Redis.Addr = v
	}
	if v, ok := lookup(EnvRedisPassword); ok {
		// Allow explicit empty override for password-less Redis.
		cfg.Redis.Password = v
	}
	if v, ok := lookup(EnvRedisDB); ok && strings.TrimSpace(v) != "" {
		db, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("%s must be an integer", EnvRedisDB)
		}
		cfg.Redis.DB = db
	}
	if v, ok := lookup(EnvLogLevel); ok && strings.TrimSpace(v) != "" {
		cfg.Log.Level = v
	}
	if v, ok := lookup(EnvAppEnvironment); ok && strings.TrimSpace(v) != "" {
		cfg.Auth.Environment = v
	}
	if v, ok := lookup(EnvJWTSecret); ok && strings.TrimSpace(v) != "" {
		cfg.Auth.JWTSecret = v
	}
	if v, ok := lookup(EnvJWTIssuer); ok && strings.TrimSpace(v) != "" {
		cfg.Auth.JWTIssuer = v
	}
	if v, ok := lookup(EnvJWTAudience); ok && strings.TrimSpace(v) != "" {
		cfg.Auth.JWTAudience = v
	}
	if v, ok := lookup(EnvSMTPMasterKey); ok {
		// Allow empty override meaning "no master key configured".
		cfg.User.SMTPMasterKey = v
	}
	return nil
}

// Validate checks required fields and positive timeout/pool bounds.
func (c Config) Validate() error {
	var errs []string

	if strings.TrimSpace(c.Server.Addr) == "" {
		errs = append(errs, "server.addr is required")
	}
	for _, item := range []struct {
		name string
		d    time.Duration
	}{
		{"server.read_header_timeout", c.Server.ReadHeaderTimeout},
		{"server.read_timeout", c.Server.ReadTimeout},
		{"server.write_timeout", c.Server.WriteTimeout},
		{"server.idle_timeout", c.Server.IdleTimeout},
		{"server.shutdown_timeout", c.Server.ShutdownTimeout},
	} {
		if item.d <= 0 {
			errs = append(errs, item.name+" must be greater than zero")
		}
	}

	errs = append(errs, validatePostgres("database.primary", c.Database.Primary)...)
	errs = append(errs, validatePostgres("database.log", c.Database.Log)...)
	errs = append(errs, validateRedis(c.Redis)...)
	errs = append(errs, validateAuth(c.Auth)...)
	if len(c.User.SMTPMasterKey) != 0 && len(c.User.SMTPMasterKey) != 32 {
		errs = append(errs, "user.smtp_master_key must be exactly 32 bytes when configured")
	}

	switch strings.ToLower(strings.TrimSpace(c.Log.Level)) {
	case "debug", "info", "warn", "error":
	default:
		errs = append(errs, "log.level must be one of debug, info, warn, error")
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func validatePostgres(prefix string, cfg PostgresConfig) []string {
	var errs []string
	if strings.TrimSpace(cfg.DSN) == "" {
		errs = append(errs, prefix+".dsn is required")
	}
	for _, item := range []struct {
		name string
		d    time.Duration
	}{
		{prefix + ".conn_max_lifetime", cfg.ConnMaxLifetime},
		{prefix + ".conn_max_idle_time", cfg.ConnMaxIdleTime},
		{prefix + ".startup_timeout", cfg.StartupTimeout},
		{prefix + ".ping_timeout", cfg.PingTimeout},
	} {
		if item.d <= 0 {
			errs = append(errs, item.name+" must be greater than zero")
		}
	}
	if cfg.MaxOpenConns <= 0 {
		errs = append(errs, prefix+".max_open_conns must be greater than zero")
	}
	if cfg.MaxIdleConns < 0 {
		errs = append(errs, prefix+".max_idle_conns must be zero or greater")
	}
	if cfg.MaxIdleConns > cfg.MaxOpenConns {
		errs = append(errs, prefix+".max_idle_conns must not exceed "+prefix+".max_open_conns")
	}
	return errs
}

func validateRedis(cfg RedisConfig) []string {
	var errs []string
	if strings.TrimSpace(cfg.Addr) == "" {
		errs = append(errs, "redis.addr is required")
	}
	if cfg.DB < 0 {
		errs = append(errs, "redis.db must be zero or greater")
	}
	if cfg.PoolSize <= 0 {
		errs = append(errs, "redis.pool_size must be greater than zero")
	}
	for _, item := range []struct {
		name string
		d    time.Duration
	}{
		{"redis.dial_timeout", cfg.DialTimeout},
		{"redis.read_timeout", cfg.ReadTimeout},
		{"redis.write_timeout", cfg.WriteTimeout},
		{"redis.startup_timeout", cfg.StartupTimeout},
		{"redis.ping_timeout", cfg.PingTimeout},
	} {
		if item.d <= 0 {
			errs = append(errs, item.name+" must be greater than zero")
		}
	}
	return errs
}

func validateAuth(cfg AuthConfig) []string {
	var errs []string
	env := strings.ToLower(strings.TrimSpace(cfg.Environment))
	switch env {
	case "development", "production":
	default:
		errs = append(errs, "auth.environment must be development or production")
	}
	if strings.TrimSpace(cfg.JWTSecret) == "" {
		errs = append(errs, "auth.jwt_secret is required")
	} else if len(cfg.JWTSecret) < 32 {
		errs = append(errs, "auth.jwt_secret must be at least 32 characters")
	}
	if strings.TrimSpace(cfg.JWTIssuer) == "" {
		errs = append(errs, "auth.jwt_issuer is required")
	}
	if strings.TrimSpace(cfg.JWTAudience) == "" {
		errs = append(errs, "auth.jwt_audience is required")
	}
	if strings.TrimSpace(cfg.RefreshCookieName) == "" {
		errs = append(errs, "auth.refresh_cookie_name is required")
	}
	if strings.TrimSpace(cfg.RefreshCookiePath) == "" {
		errs = append(errs, "auth.refresh_cookie_path is required")
	}
	if env == "production" && !cfg.RefreshCookieSecure {
		errs = append(errs, "auth.refresh_cookie_secure must be true in production")
	}
	if cfg.BcryptCost < 12 {
		errs = append(errs, "auth.bcrypt_cost must be at least 12")
	}
	if len(cfg.TrustedOrigins) == 0 {
		errs = append(errs, "auth.trusted_origins must contain at least one origin")
	}
	for _, origin := range cfg.TrustedOrigins {
		if strings.TrimSpace(origin) == "" {
			errs = append(errs, "auth.trusted_origins entries must be non-empty")
			break
		}
	}
	return errs
}
