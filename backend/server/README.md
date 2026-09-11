# Server

`server/` contains the Go API service. `server/admin/` is a separate management
frontend and is not part of the Go module's implementation scope.

## Quick start

1. Ensure local Docker middleware is healthy (see `docs/data-stores.md`).

2. Copy the example config and replace secrets from the ignored
   `configs/infrastructure.local.env` values:

```sh
cp configs/config.example.toml configs/config.local.toml
# Edit DSNs / Redis settings, or export PRIMARY_POSTGRES_DSN, LOG_POSTGRES_DSN,
# REDIS_ADDR, REDIS_PASSWORD, and REDIS_DB before starting.
```

3. Start the server (single entry point; automatically applies primary and
   log migrations, bootstraps the initial administrator and super_admin role
   on a fresh database, and fails closed — no HTTP listener — when config is
   invalid, any required store is down, migrations fail, or bootstrap fails):

```sh
go run ./cmd/server -config configs/config.local.toml
# or
make run
```

4. Probe the bootstrap endpoints:

```sh
curl -sS http://127.0.0.1:8080/healthz
curl -sS http://127.0.0.1:8080/readyz
```

## Configuration

Configuration is loaded once at startup.

Precedence:

1. TOML file selected by `-config`
2. Named environment overrides

| Environment variable | Overrides |
| --- | --- |
| `SERVER_ADDR` | `server.addr` |
| `PRIMARY_POSTGRES_DSN` | `database.primary.dsn` |
| `LOG_POSTGRES_DSN` | `database.log.dsn` |
| `REDIS_ADDR` | `redis.addr` |
| `REDIS_PASSWORD` | `redis.password` |
| `REDIS_DB` | `redis.db` |
| `LOG_LEVEL` | `log.level` |
| `APP_ENVIRONMENT` | `auth.environment` |
| `JWT_SECRET` | `auth.jwt_secret` |
| `JWT_ISSUER` | `auth.jwt_issuer` |
| `JWT_AUDIENCE` | `auth.jwt_audience` |
| `SMTP_MASTER_KEY` | `user.smtp_master_key` |

`bootstrap-admin` credentials come from `BOOTSTRAP_ADMIN_PASSWORD` (username
is fixed to `admin`; `BOOTSTRAP_ADMIN_USERNAME` may only restate it and any
other value aborts startup) and optional `BOOTSTRAP_ADMIN_DISPLAY_NAME`;
development defaults apply only when the environment is `development` and no
administrator exists yet.

`POSTGRES_DSN` is not supported. Use `PRIMARY_POSTGRES_DSN` and `LOG_POSTGRES_DSN`.

Committed example keys in `configs/config.example.toml`:

| Key | Purpose |
| --- | --- |
| `server.addr` | HTTP listen address |
| `server.read_header_timeout` | `http.Server.ReadHeaderTimeout` |
| `server.read_timeout` | `http.Server.ReadTimeout` |
| `server.write_timeout` | `http.Server.WriteTimeout` |
| `server.idle_timeout` | `http.Server.IdleTimeout` |
| `server.shutdown_timeout` | Graceful shutdown bound |
| `database.primary.*` | Primary PostgreSQL DSN, pool, startup/ping timeouts |
| `database.log.*` | Log PostgreSQL DSN, pool, startup/ping timeouts |
| `redis.addr` / `password` / `db` | Redis endpoint and database index |
| `redis.pool_size` | Redis connection pool size |
| `redis.dial_timeout` / `read_timeout` / `write_timeout` | Redis I/O timeouts |
| `redis.startup_timeout` / `ping_timeout` | Redis startup and readiness bounds |
| `log.level` | `debug`, `info`, `warn`, or `error` |
| `auth.user_jwt_audience` | Business-user JWT audience (default `easy-admin-user`) |
| `auth.user_refresh_cookie_name` / `_path` / `_secure` | Business-user refresh cookie (default `ea_user_refresh`, `/api/v1/auth`) |
| `user.smtp_master_key` | Exactly 32 bytes; encrypts/decrypts stored SMTP passwords at the application boundary. Empty means SMTP secrets cannot be configured and email verification stays unavailable |
| `user.verification_token_ttl` | Bounded TTL for one-time email verification tokens (default and maximum 24h) |

Local files matching `configs/config.local.toml` and
`configs/infrastructure.local.env` are gitignored.

Runtime diagnostics are JSON `slog` records on stdout. The log PostgreSQL store
is reserved for future durable audit/operation records and is never used as a
runtime log sink by this foundation.

## Endpoints

| Method | Path | Behavior |
| --- | --- | --- |
| `GET` | `/healthz` | Liveness; does not call PostgreSQL or Redis |
| `GET` | `/readyz` | Readiness; pings primary PostgreSQL, log PostgreSQL, and Redis. Any failure returns HTTP 503 / code `50001` |

Both endpoints return the shared envelope documented in `docs/openapi.yaml`.
A valid inbound `X-Request-ID` is echoed unchanged. Validity means 1-128
characters of ASCII alphanumeric plus `.`, `_`, `:`, or `-`. Empty, oversized,
whitespace-containing, Unicode, or otherwise invalid values are replaced with a
newly generated compliant ID that appears in both the response header and body.

## Development commands

```sh
make fmt
make test
make vet
make build
make check
```

Equivalent Go commands:

```sh
gofmt -w $(find . -name '*.go' -not -path './admin/*')
go test ./...
go vet ./...
go build ./...
```

## Design documents

- [Architecture](docs/architecture.md)
- [Data stores](docs/data-stores.md)
- [Coding standards](docs/coding-standards.md)
- [OpenAPI](docs/openapi.yaml)
- [Contributor and agent instructions](AGENTS.md)


## Startup, migrations, and bootstrap

There is one startup entry point, `cmd/server`. On every start it performs, in
order and fail-closed:

1. Configuration load and validation (`-config` TOML, then environment overrides).
2. Primary PostgreSQL connectivity check.
3. Primary migrations from `migrations/primary/*.up.sql` (explicit SQL, tracked
   in `schema_migrations`, idempotent: already-applied versions are skipped).
4. Log PostgreSQL connectivity check.
5. Log migrations from `migrations/log/*.up.sql` (same mechanism).
6. Idempotent bootstrap (see below).
7. Dependency wiring (Redis, JWT, session stores, services).
8. HTTP listener startup with graceful shutdown on SIGINT/SIGTERM.

If any step fails, the process exits non-zero and never listens on the HTTP
port. `cmd/migrate` and `cmd/bootstrap-admin` no longer exist; migration and
bootstrap always run inside the startup path. `AutoMigrate` is never used.

Bootstrap is idempotent and data preserving:

- The built-in `super_admin` role is guaranteed to exist; its name and
  description are kept at “超级管理员” (updated in place when they differ;
  repeated startups are no-ops).
- When the primary database contains no administrator at all, the initial
  account `admin` is created (username fixed; `BOOTSTRAP_ADMIN_USERNAME` may
  only restate `admin`) from `BOOTSTRAP_ADMIN_PASSWORD` (development default
  when unset in `development`) with the `super_admin` role and display name
  “超级管理员”. When administrators already exist, nothing is created or
  modified and no credentials are required.
- No administrator, role, permission, or business data is ever deleted by
  startup.

Development bootstrap defaults (`admin` / `admin123`) are allowed only when
`auth.environment = development` and the database has no administrator yet.
Production requires non-default credentials via `BOOTSTRAP_ADMIN_PASSWORD`
(the username is always the fixed `admin` account). Ordinary password
create/change/reset still enforce
8-72 byte length (UTF-8 bytes).

## Admin sessions and revocation

Administrator login creates one Redis session. Keys and rules:

| Key | Purpose | TTL | Cleanup / failure policy |
| --- | --- | --- | --- |
| `easy-admin:admin-session:<sid>` | JSON session: admin id, sha-256 of the current refresh JTI, `active`/`revoked` status, `auth_epoch`, expiry | capped to refresh expiry (30 days) | logout marks revoked; replay marks revoked; password/account transitions remove it best-effort |
| `easy-admin:admin-session-index:<admin_id>` | SET of that administrator's session ids for best-effort full revocation | same TTL, refreshed on rotation | deleted after successful full revocation; never authoritative |

Refresh rotation is atomic (WATCH + MULTI). Reuse of a consumed refresh verifier
revokes the whole session and raises a durable `auth.refresh_replay` audit event.
Protected operations fail closed when the session is missing, revoked, corrupt,
expired, Redis is unavailable, or the stored epoch does not match primary storage.
A session has one absolute refresh expiry set at login (30 days): every rotated
refresh JWT is minted with `exp` equal to that same instant, rotation never
changes the Redis absolute expiry or grows its TTL, and the refresh-cookie
`Max-Age` reflects the remaining lifetime, so repeated refreshes can never slide
a session forward indefinitely.

Every password change, password reset, and account enable/disable increments the
administrator's `auth_epoch` column in primary PostgreSQL in the same write. Each
issued session records the epoch it was created under; the authentication
middleware and refresh flow compare it to the current primary value on every
protected request. This is the authoritative, race-safe invalidation: even a
session minted concurrently by a login that Redis revocation never observed can
never be used after the transition. Re-enabling an account bumps the epoch again
and never restores pre-disable sessions. Redis full-session revocation is kept as
best-effort hygiene and is never the security boundary.

## Roles and permissions

Authorization is resolved from primary PostgreSQL on every protected request; JWT
claims never carry roles or permissions. Disabled roles are excluded from role
identity (`/me` role codes), the `super_admin` bypass check, effective permission
union, and role assignment lookups, so disabling a role takes effect on the very
next request without waiting for access-token expiry. The built-in `super_admin`
role cannot be disabled; other built-in and custom roles can.

## Audit failure protocol

Management writes and mandatory security events follow the PRD pending-first
protocol. The initial `pending` audit insert happens before the primary write:
if it fails, the operation returns `audit unavailable` (503 / 50002) and the
security-sensitive write never runs. A failed final `succeeded`/`failed` outcome
update leaves a diagnosable `pending` row and emits a sanitized high-severity
runtime log; the two databases cannot be updated in one distributed transaction.
Successful login creates its Redis session first and revokes it again if the final
audit cannot be persisted, so an unaudited successful login is never returned.
Failed logins return one uniform response whether or not their audit can be
persisted and never reveal whether a username exists. A login for a nonexistent
username still executes one configured-cost bcrypt comparison against a dummy
hash created once at service startup, so unknown-user and wrong-password attempts
cost equivalent work and cannot be distinguished by timing.

## Business users (USR-001)

Business users are a separate identity domain from administrators. They never
reuse the `administrators` table, the administrator JWT audience, refresh cookie,
Redis session namespace, roles, or permissions.

### Public endpoints

| Method | Path | Behavior |
| --- | --- | --- |
| `GET` | `/api/v1/public/settings` | Safe whitelist only: platform name, public URLs, registration switch, enabled login methods, verification requirement, default avatar |
| `POST` | `/api/v1/auth/register` | Username + email + password. Active immediately when verification is disabled, otherwise `pending_verification` plus a one-time email link |
| `POST` | `/api/v1/auth/verify-email` | Redeems the one-time token (single-use, expiring, replay-safe) |
| `POST` | `/api/v1/auth/resend-verification` | Invalidates the previous token, issues a fresh one, rate-limited, no existence disclosure |
| `POST` | `/api/v1/auth/login` | One `identifier` field plus password; `@` means email when email login is enabled, otherwise username |
| `POST` | `/api/v1/auth/refresh` | Rotates the refresh verifier inside the session's absolute expiry |
| `POST` | `/api/v1/auth/logout` | Authenticated; revokes the current session (pending-first audit) |
| `GET` | `/api/v1/me` | Authenticated profile with exact four-decimal point strings |

### Client contract (no client ships in this repository)

- Access token: `Authorization: Bearer <access_token>`; 15 minutes; audience
  `easy-admin-user`; payload carries only subject, session id, and token type.
- Refresh token: `ea_user_refresh` HttpOnly, SameSite=Strict, Path `/api/v1/auth`,
  `Secure` whenever `auth.environment = production`. Never readable from script.
- A future client must be given its API origin from deployment/runtime
  configuration; the database `public_*_url` values are link/email metadata and
  never reconfigure the running server.
- Cross-site refresh requires the request `Origin` to appear in
  `auth.trusted_origins`; a disallowed origin fails closed.
- `POST /api/v1/auth/refresh` returns a new access token and rotates the cookie;
  the client must reload `/me` afterwards because refresh never returns a profile
  and never updates `last_login_at`.

### Sessions, verification, and rate limits

User sessions live in `easy-admin:user-session:<sid>` with an index set in
`easy-admin:user-session-index:<user_id>`; both are capped to the session's
absolute refresh expiry (30 days) and rotation never extends it. Reusing a
consumed refresh verifier revokes the whole session and raises
`user.refresh_replay`. Disabling an account or resetting a password bumps
`users.auth_epoch`, which the middleware compares against the session on every
protected request. Missing, revoked, corrupt, expired, or Redis-unavailable
sessions always fail closed.

Verification tokens are 32 random bytes rendered as 64 hex characters, stored
only as a SHA-256 digest in `easy-admin:user-verify:<user_id>` with a bounded TTL
(`user.verification_token_ttl`, default 24h). Redemption is an atomic Lua
compare-and-delete, so a token is single-use and a replay is rejected with the
same generic error as an unknown or expired token. Resend replaces the stored
digest, invalidating the previous link.

Fixed-window Redis throttles cover registration (5/hour/IP), login (10/15min per
IP and per hashed identifier), resend (3/hour/IP), and verification (10/hour/IP).
A Redis throttle failure is treated as "limited" rather than as permission.

### Points and levels

`points_balance` and `consumption_points` are `NUMERIC(20,4)` and travel as
fixed four-decimal strings (for example `"123.4567"`); no code path uses binary
floating point. Every change is one immutable `user_point_transactions` row with
a required reason, the acting administrator, both resulting balances, and a unique
idempotency key; the ledger insert and the locked user-row update share one
primary-database transaction. Available points can never go negative, cumulative
consumption points can never decrease, and replaying an idempotency key returns a
`40015` conflict without applying the change twice.

Automatic levels take the highest enabled `user_levels.threshold_points` that is
less than or equal to cumulative consumption points, recalculated in the same
transaction as any consumption change. A manual assignment remains authoritative
across later point changes until an administrator switches the user back to
`auto`, which recalculates immediately.

### System settings and SMTP secrets

`system_settings` is one typed singleton row (`id = 1`) with an optimistic-lock
`version`: concurrent writers get `40013` instead of silent last-write-wins. At
least one login method must stay enabled, a disabled level cannot become the
default, and enabling email verification requires an enabled, validated,
decryptable SMTP configuration with `starttls` or `ssl`.

SMTP passwords are encrypted with AES-256-GCM using a random nonce per message
under the 32-byte `SMTP_MASTER_KEY`. Administrator reads return only
`password_configured`; a blank `smtp_password` on update retains the stored
secret. Plaintext and ciphertext never appear in any API response, audit detail,
runtime log, test output, or CLI output.

### Administrator endpoints

Under `/api/v1/admin`: `GET|POST /users`, `GET|PATCH /users/{id}`,
`POST /users/{id}/{enable,disable,reset-password,points-adjust,level}`,
`GET /users/{id}/point-transactions`, `GET|POST /user-levels`,
`GET|PATCH /user-levels/{id}`, and `GET|PUT /system-settings`. The generic
`PATCH /users/{id}` accepts only email, nickname, avatar URL, and remark;
username, registration IP/time, points, level mode, status, and password have
dedicated operations. Every mutation re-checks backend permissions from primary
PostgreSQL and writes a pending-first audit record.
