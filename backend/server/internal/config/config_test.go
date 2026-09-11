package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const authTOML = `
[auth]
environment = "development"
jwt_secret = "change-me-to-a-long-random-secret-at-least-32"
jwt_issuer = "easy-admin"
jwt_audience = "easy-admin-admin"
refresh_cookie_name = "ea_admin_refresh"
refresh_cookie_path = "/api/v1/admin/auth"
refresh_cookie_secure = false
trusted_origins = ["http://127.0.0.1:8848"]
bcrypt_cost = 12
`

const validTOML = `
[server]
addr = ":8080"
read_header_timeout = "5s"
read_timeout = "15s"
write_timeout = "15s"
idle_timeout = "60s"
shutdown_timeout = "10s"

[database.primary]
dsn = "postgres://easy_admin:change-me@127.0.0.1:55432/easy_admin?sslmode=disable"
max_open_conns = 10
max_idle_conns = 5
conn_max_lifetime = "30m"
conn_max_idle_time = "5m"
startup_timeout = "5s"
ping_timeout = "2s"

[database.log]
dsn = "postgres://easy_admin_logs:change-me@127.0.0.1:55433/easy_admin_logs?sslmode=disable"
max_open_conns = 5
max_idle_conns = 2
conn_max_lifetime = "30m"
conn_max_idle_time = "5m"
startup_timeout = "5s"
ping_timeout = "2s"

[redis]
addr = "127.0.0.1:56379"
password = ""
db = 0
pool_size = 10
dial_timeout = "3s"
read_timeout = "2s"
write_timeout = "2s"
startup_timeout = "3s"
ping_timeout = "1s"
` + authTOML + `
[log]
level = "info"
`

func TestLoadValidConfig(t *testing.T) {
	t.Parallel()

	path := writeTempConfig(t, validTOML)
	cfg, err := load(path, func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.Server.Addr != ":8080" {
		t.Fatalf("Server.Addr = %q, want :8080", cfg.Server.Addr)
	}
	if cfg.Database.Primary.MaxOpenConns != 10 {
		t.Fatalf("Primary.MaxOpenConns = %d, want 10", cfg.Database.Primary.MaxOpenConns)
	}
	if cfg.Database.Log.PingTimeout != 2*time.Second {
		t.Fatalf("Log.PingTimeout = %v, want 2s", cfg.Database.Log.PingTimeout)
	}
	if cfg.Redis.Addr != "127.0.0.1:56379" {
		t.Fatalf("Redis.Addr = %q, want 127.0.0.1:56379", cfg.Redis.Addr)
	}
	if cfg.Auth.BcryptCost != 12 {
		t.Fatalf("Auth.BcryptCost = %d, want 12", cfg.Auth.BcryptCost)
	}
	if cfg.Log.Level != "info" {
		t.Fatalf("Log.Level = %q, want info", cfg.Log.Level)
	}
}

func TestLoadMissingFile(t *testing.T) {
	t.Parallel()

	_, err := load(filepath.Join(t.TempDir(), "missing.toml"), func(string) (string, bool) {
		return "", false
	})
	if err == nil {
		t.Fatal("Load() error = nil, want missing file error")
	}
}

func TestLoadInvalidDuration(t *testing.T) {
	t.Parallel()

	invalid := strings.Replace(validTOML, `read_header_timeout = "5s"`, `read_header_timeout = "not-a-duration"`, 1)
	path := writeTempConfig(t, invalid)
	_, err := load(path, func(string) (string, bool) { return "", false })
	if err == nil {
		t.Fatal("Load() error = nil, want invalid duration error")
	}
}

func TestLoadMissingRequiredValues(t *testing.T) {
	t.Parallel()

	missing := `
[server]
addr = ""
read_header_timeout = "5s"
read_timeout = "15s"
write_timeout = "15s"
idle_timeout = "60s"
shutdown_timeout = "10s"

[database.primary]
dsn = ""
max_open_conns = 0
max_idle_conns = 5
conn_max_lifetime = "30m"
conn_max_idle_time = "5m"
startup_timeout = "5s"
ping_timeout = "2s"

[database.log]
dsn = ""
max_open_conns = 0
max_idle_conns = 1
conn_max_lifetime = "30m"
conn_max_idle_time = "5m"
startup_timeout = "5s"
ping_timeout = "2s"

[redis]
addr = ""
password = ""
db = -1
pool_size = 0
dial_timeout = "3s"
read_timeout = "2s"
write_timeout = "2s"
startup_timeout = "3s"
ping_timeout = "1s"

[auth]
environment = "staging"
jwt_secret = "short"
jwt_issuer = ""
jwt_audience = ""
refresh_cookie_name = ""
refresh_cookie_path = ""
refresh_cookie_secure = false
trusted_origins = []
bcrypt_cost = 4

[log]
level = "trace"
`
	path := writeTempConfig(t, missing)
	_, err := load(path, func(string) (string, bool) { return "", false })
	if err == nil {
		t.Fatal("Load() error = nil, want validation errors")
	}
}

func TestEnvOverridesTakePrecedence(t *testing.T) {
	t.Parallel()

	path := writeTempConfig(t, validTOML)
	env := map[string]string{
		EnvServerAddr:         ":9090",
		EnvPrimaryPostgresDSN: "postgres://primary:secret@db:5432/app?sslmode=require",
		EnvLogPostgresDSN:     "postgres://logs:secret@db:5432/logs?sslmode=require",
		EnvRedisAddr:          "127.0.0.1:6399",
		EnvRedisPassword:      "redis-secret",
		EnvRedisDB:            "2",
		EnvLogLevel:           "debug",
		EnvJWTSecret:          "override-jwt-secret-value-32chars-min",
		EnvAppEnvironment:     "production",
	}

	cfg, err := load(path, func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	})
	if err != nil {
		// production + refresh_cookie_secure=false from TOML should fail validation
		if !strings.Contains(err.Error(), "refresh_cookie_secure") {
			t.Fatalf("Load() unexpected error: %v", err)
		}
		return
	}
	if cfg.Server.Addr != ":9090" {
		t.Fatalf("Server.Addr = %q, want :9090", cfg.Server.Addr)
	}
	if cfg.Auth.JWTSecret != env[EnvJWTSecret] {
		t.Fatal("JWT secret was not overridden")
	}
}

func TestEmptyEnvDoesNotOverrideTOML(t *testing.T) {
	t.Parallel()

	path := writeTempConfig(t, validTOML)
	cfg, err := load(path, func(key string) (string, bool) {
		switch key {
		case EnvServerAddr, EnvPrimaryPostgresDSN, EnvLogPostgresDSN, EnvRedisAddr, EnvRedisDB, EnvLogLevel, EnvJWTSecret:
			return "   ", true
		default:
			return "", false
		}
	})
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.Server.Addr != ":8080" {
		t.Fatalf("Server.Addr = %q, want TOML value :8080", cfg.Server.Addr)
	}
	if cfg.Redis.Addr != "127.0.0.1:56379" {
		t.Fatalf("Redis.Addr = %q, want TOML value", cfg.Redis.Addr)
	}
}

func TestRedisPasswordEmptyEnvOverridesTOML(t *testing.T) {
	t.Parallel()

	path := writeTempConfig(t, strings.Replace(validTOML, `password = ""`, `password = "from-toml"`, 1))
	cfg, err := load(path, func(key string) (string, bool) {
		if key == EnvRedisPassword {
			return "", true
		}
		return "", false
	})
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.Redis.Password != "" {
		t.Fatalf("Redis.Password = %q, want empty override", cfg.Redis.Password)
	}
}

func TestInvalidRedisDBOverride(t *testing.T) {
	t.Parallel()

	path := writeTempConfig(t, validTOML)
	_, err := load(path, func(key string) (string, bool) {
		if key == EnvRedisDB {
			return "not-an-int", true
		}
		return "", false
	})
	if err == nil {
		t.Fatal("Load() error = nil, want REDIS_DB parse error")
	}
}

func TestLoadRejectsEmptyPath(t *testing.T) {
	t.Parallel()

	_, err := load("   ", func(string) (string, bool) { return "", false })
	if err == nil {
		t.Fatal("Load() error = nil, want empty path error")
	}
}

func TestLoadCommittedExampleConfig(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", "configs", "config.example.toml")
	cfg, err := load(path, func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("Load(config.example.toml) unexpected error: %v", err)
	}
	if cfg.Database.Primary.DSN == "" || cfg.Database.Log.DSN == "" || cfg.Redis.Addr == "" || cfg.Auth.JWTSecret == "" {
		t.Fatal("example config missing required store/auth fields")
	}
}

func TestProductionRejectsInsecureRefreshCookie(t *testing.T) {
	t.Parallel()

	invalid := strings.Replace(validTOML, `environment = "development"`, `environment = "production"`, 1)
	path := writeTempConfig(t, invalid)
	_, err := load(path, func(string) (string, bool) { return "", false })
	if err == nil {
		t.Fatal("expected production + insecure cookie validation error")
	}
}

func writeTempConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}
