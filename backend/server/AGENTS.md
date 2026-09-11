# Server Instructions

These instructions apply to all files under `server/`, except `server/admin/`.
The `server/admin/` directory is a separate backend-management frontend and is
outside the Go service scope unless a task explicitly includes it.

## Priorities

1. Keep the code easy to trace from route to handler to service to repository.
2. Add a layer or interface only when it owns real behavior or creates a real
   test/substitution boundary.
3. Prefer small packages and explicit dependency wiring over framework-style
   containers, global state, or reflection.
4. Keep transport DTOs, domain values, and database models separate when their
   responsibilities differ. Do not duplicate types without a concrete reason.

## Required Backend Stack

- Go, one module rooted at `server/`
- Gin for HTTP routing
- GORM with PostgreSQL for persistence
- A separate PostgreSQL connection and database for durable audit/operation logs
- Redis for explicitly approved shared ephemeral state
- TOML for file configuration
- Standard library `log/slog` for structured logging
- OpenAPI at `server/docs/openapi.yaml` as the API source of truth

Runtime diagnostics remain structured `slog` output to stdout. Do not write
runtime logs to either PostgreSQL database. The log database is reserved for
durable audit and operation records with an explicit schema and retention rule.
Redis is approved infrastructure, but each key still needs a documented purpose,
namespace, TTL, invalidation rule, and failure policy before use. Administrator
JWT authentication and RBAC are implemented under `/api/v1/admin` per SRV-003.

## Package Boundaries

- `cmd/server`: process startup, dependency wiring, signals, and shutdown only.
- `internal/config`: load and validate configuration once at startup.
- `internal/domain`: business entities, invariants, and domain errors.
- `internal/service`: use cases and transaction boundaries.
- `internal/repository`: persistence contracts and implementations.
- `internal/transport/http`: Gin routes, middleware, request DTOs, response DTOs,
  validation, and mapping errors to HTTP responses.
- `internal/platform`: primary PostgreSQL, log PostgreSQL, Redis, and other
  external integrations. Keep each client's lifecycle explicit.

Create `domain`, `service`, and `repository` packages only when the first real
feature needs them. Empty placeholder directories are prohibited.

## Non-Negotiable Rules

- Pass `context.Context` into database and other blocking operations. Never pass
  `*gin.Context` beyond the HTTP transport layer.
- Validate input at the transport boundary and keep handlers thin.
- Keep SQL/GORM details in repositories, not handlers or services.
- Use explicit SQL migrations for schema changes. Do not use `AutoMigrate` in
  application startup or as the production migration strategy.
- Never write business records to the log database or audit/operation records to
  the primary database. Do not attempt transactions across the two databases.
- Use transactions for invariants involving multiple writes.
- Never expose internal errors, SQL details, stack traces, credentials, or
  tokens in API responses or logs.
- Configure HTTP timeouts and graceful shutdown. Close resources owned by the
  process.
- Return the shared JSON envelope and use semantic HTTP status codes.
- Serialize persistent IDs as decimal strings and timestamps as UTC RFC 3339.
- Do not introduce microservices, CQRS, event buses, dependency-injection
  frameworks, generic base repositories, or speculative shared packages.

## Verification

For every Go change, run:

```sh
gofmt -w <changed-go-files>
go test ./...
go vet ./...
go build ./...
```

Run `go test -race ./...` for concurrency, background-worker, shared-state, or
Redis coordination changes. Report unavailable integration services and skipped
checks explicitly.
