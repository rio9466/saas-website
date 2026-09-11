# Go Coding Standards

## Readability

- Prefer straightforward Go over clever helpers. Code should read top to bottom.
- Use short functions with one responsibility, but do not split code solely to
  reduce line count.
- Name packages for what they provide, not generic buckets such as `common`,
  `utils`, `base`, or `manager`.
- Avoid stutter: use `config.Load`, not `config.LoadConfig`.
- Keep exported APIs small and document exported identifiers whose purpose is
  not obvious from the name and type.

## Dependencies and Errors

- Pass dependencies through constructors; do not use mutable package globals.
- Define an interface where it is consumed and only for a real test or
  substitution boundary.
- Return errors; do not panic for expected runtime failures.
- Wrap errors with context using `%w`. Compare known errors with `errors.Is` or
  `errors.As`.
- Log an error once at the process/transport boundary. Lower layers add context
  but generally do not log and return the same error.

## HTTP

- Handlers bind and validate input, call one use case, then format a response.
- Use a stable envelope:

```json
{
  "code": 0,
  "message": "success",
  "data": {},
  "request_id": "request-id"
}
```

- Use `code: 0` for success and registered nonzero application codes for errors.
- Use semantic HTTP status codes; do not turn every result into HTTP 200.
- Error-code ranges are: `10000-19999` validation/protocol, `20000-29999`
  authentication/session, `30000-39999` authorization, `40000-49999` domain,
  and `50000-59999` dependency/internal.
- Keep response messages safe for clients. Preserve detailed causes only in
  internal errors and sanitized logs.
- Use one-based `page` and `page_size`; defaults are 1 and 20, with a maximum of
  100 unless OpenAPI specifies otherwise.

## Data

- Keep request/response DTOs separate from GORM models when their shapes or
  responsibilities differ.
- Serialize persistent integer IDs as decimal strings in JSON.
- Return timestamps as UTC RFC 3339 strings with `_at` field names.
- Do not issue unbounded list queries, accidental full-table updates/deletes, or
  string-built SQL.
- Use explicit transactions for multi-write invariants.

## Configuration and Security

- Commit only `config.example.toml`; ignore local config and secret files.
- Read configuration once during startup and validate required values.
- Never log passwords, tokens, connection strings, or sensitive request bodies.
- Set read-header, read, write, idle, startup, dependency, and shutdown timeouts.
- Add CORS, authentication, rate limiting, and Redis key usage only with an
  approved need and documented behavior. Redis client wiring alone does not
  approve business keys.

## Tests

- Prefer table-driven tests when several inputs exercise the same behavior.
- Use `httptest` for router, middleware, handler, and response tests.
- Depend on a small consumed interface to test dependency success and failure.
- Use real PostgreSQL integration tests for behavior that depends on PostgreSQL
  semantics; do not pretend an in-memory database is equivalent.
- Every bug fix should include a regression test when practical.

## Tooling

The minimum completion gate is:

```sh
gofmt -w <changed-go-files>
go test ./...
go vet ./...
go build ./...
```

Keep the Makefile as readable aliases for these commands, not as a second build
system. Avoid generated mocks and additional linters in the bootstrap unless a
real need justifies them.
