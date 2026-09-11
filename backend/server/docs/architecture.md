# Server Architecture

## Goal

Build one understandable Go service for a small-to-medium product. A developer
should be able to follow one request without learning an internal framework.
The design allows features to be added cleanly but does not create empty layers
for hypothetical future work.

## Directory Shape

The foundation creates only directories that contain working code:

```text
server/
├── cmd/server/main.go
├── configs/config.example.toml
├── docs/
│   ├── architecture.md
│   ├── coding-standards.md
│   ├── data-stores.md
│   └── openapi.yaml
├── internal/
│   ├── config/
│   ├── platform/
│   │   ├── postgres/
│   │   └── redis/
│   └── transport/http/
│       ├── handler/
│       ├── middleware/
│       └── response/
├── .gitignore
├── AGENTS.md
├── Makefile
├── README.md
├── go.mod
└── go.sum
```

When the first business feature arrives, add packages only as needed:

```text
internal/
├── domain/<feature>/
├── service/<feature>/
└── repository/<feature>/
```

## Request Flow

```text
HTTP request
  -> Gin middleware
  -> handler (bind and validate)
  -> service (business rule)
  -> repository (persistence)
  -> PostgreSQL
```

Responses travel back through the same boundaries. Handlers translate domain
results and errors into the documented HTTP status and JSON envelope. Services
do not know about Gin, and repositories do not decide HTTP behavior.

## Startup and Shutdown

`cmd/server/main.go` is the single startup entry point. It performs explicit
wiring in this order, fail-closed (no HTTP listener when any step fails):

1. Parse the config path and load validated TOML configuration.
2. Create the structured logger (`slog` to stdout).
3. Open primary PostgreSQL and verify connectivity with a bounded startup timeout.
4. Apply primary migrations from `migrations/primary/*.up.sql` (explicit SQL,
   tracked in `schema_migrations`, idempotent).
5. Open log PostgreSQL and verify connectivity with a bounded startup timeout.
6. Apply log migrations from `migrations/log/*.up.sql` (same mechanism).
7. Run the idempotent bootstrap: guarantee the built-in `super_admin` role
   (Chinese name/description) and create the initial `admin` administrator on a
   fresh database only. Existing data is never deleted or overwritten.
8. Open Redis and verify connectivity with a bounded startup timeout.
9. Build handlers, middleware, router, and `http.Server`.
10. Start serving and wait for `SIGINT` or `SIGTERM`.
11. Gracefully stop HTTP traffic, then close Redis and both PostgreSQL clients.

The former `cmd/migrate` and `cmd/bootstrap-admin` entry points are folded into
this startup path (`internal/platform/migrate` and `internal/platform/bootstrap`).
If a later dependency fails during startup, already-opened clients are closed via
deferred cleanup. Startup fails with a useful fixed message when required
configuration or any store is unavailable. Secrets, DSNs, passwords, and driver
error text must never be logged or printed.

## Health Endpoints

- `GET /healthz`: process liveness; does not call dependencies.
- `GET /readyz`: readiness; checks primary PostgreSQL, log PostgreSQL, and Redis
  with short per-store timeouts.

Both endpoints use the common response envelope and are documented in OpenAPI.

## Authentication and Authorization

Two completely separate authentication surfaces share one process and one JWT
signing secret but never share an audience, cookie, Redis namespace, session
document, or permission model:

| Surface | Path prefix | Audience | Refresh cookie | Redis namespace | Subject store |
| --- | --- | --- | --- | --- | --- |
| Backend administrator | `/api/v1/admin/**` | `easy-admin-admin` | `ea_admin_refresh` (path `/api/v1/admin/auth`) | `easy-admin:admin-session*` | `administrators` |
| Business user | `/api/v1/auth/**`, `/api/v1/me` | `easy-admin-user` | `ea_user_refresh` (path `/api/v1/auth`) | `easy-admin:user-session*`, `easy-admin:user-verify:*` | `users` |

Each surface validates its own audience, so an administrator token is rejected
on user endpoints and a user token is rejected on administrator endpoints. Roles
and permissions are never carried in claims: they are read from primary
PostgreSQL on every protected request. `users.auth_epoch` and
`administrators.auth_epoch` are the authoritative invalidation counters.

Business-user code lives in `internal/domain/user`, `internal/repository/primary`
(`user.go`, `settings.go`), `internal/service/usersvc`, and the user handlers and
middleware under `internal/transport/http`. Administrator auth/RBAC behavior is
not reused, aliased, or refactored by the business-user surface except where the
shared primitives (`auth.TokenService`, `auth.SessionStore`, `apperr`, the audit
repository) are parameterized by audience, namespace, or action code.

## Public and Admin HTTP Surface

- `GET /api/v1/public/settings`: unauthenticated whitelist of platform identity
  and login/registration switches. It never exposes SMTP data, secrets, or the
  settings `version`.
- `POST /api/v1/auth/{register,verify-email,resend-verification,login,refresh}`
  and authenticated `POST /api/v1/auth/logout`, `GET /api/v1/me`.
- `/api/v1/admin/**`: administrator authentication, RBAC management, audit
  reads, business-user management, user-level management, and typed system
  settings.

There is no business-user client in this repository. `frontend/` is intentionally
empty; the public contract exists so a separately scoped client can consume it.

## Configuration

Configuration is loaded once. Precedence is deterministic:

1. TOML file selected by the `-config` flag.
2. Explicit environment overrides for secrets and deployment-specific values.

The committed example contains safe placeholders. The local config file is
ignored. Validation returns all actionable startup errors without printing
secret values.

## Data and Migrations

Primary PostgreSQL is the source of truth for business data. A separate
PostgreSQL instance is reserved for durable audit and operation records. Runtime
diagnostic logs remain structured stdout output and are not database records.
See [Data Stores and Local Middleware](data-stores.md) for ownership and failure
boundaries.

GORM is used for query and persistence code, but schema changes use ordered SQL
migration files. The bootstrap has no domain or audit tables, so it must not add
a fake model or an empty migration merely to prove that migrations exist.

## Future Authentication

Authentication is implemented (see the table above). Any new shared ephemeral
state must still document its key namespace, TTL, invalidation rule, and failure
policy in `docs/data-stores.md` before it is added.

## Architecture Boundaries

Keep one process and one Go module. Maintain separate owned clients for primary
PostgreSQL, log PostgreSQL, and Redis; do not build a generic data-store wrapper
that hides their different responsibilities. Split any further component only
for a real second consumer, independent deployment/ownership, material
testability or security need, or measured scale requirement.
