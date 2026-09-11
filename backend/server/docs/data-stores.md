# Data Stores and Local Middleware

## Ownership

The server uses three independent data-store clients with explicit ownership:

| Store | Purpose | Must not contain |
| --- | --- | --- |
| Primary PostgreSQL | Product and business source-of-truth data | Runtime logs, audit/operation logs |
| Log PostgreSQL | Durable audit and operation records | Product source-of-truth data |
| Redis | Approved ephemeral/shared state with a documented TTL and failure policy | Primary records or permanent audit history |

Application diagnostics remain JSON `slog` records on stdout. The log database
is not a sink for arbitrary debug, SQL, HTTP body, or stack-trace output.

The two PostgreSQL stores use separate instances, credentials, databases,
connection pools, configuration sections, migrations, and lifecycle cleanup.
Cross-database transactions are prohibited. A future audit feature must state
whether a failed audit write blocks the business action or is retried through an
approved durable mechanism; it must not silently claim atomicity across stores.

## Approved Runtime Policy

For the foundation task, all three connections are required at startup and for
readiness. This fail-fast policy prevents the service from appearing healthy
while an approved dependency is unavailable. Liveness remains dependency-free.

No table or Redis key is created until a feature defines its schema, ownership,
retention/TTL, privacy classification, and failure behavior. In particular,
SRV-002 establishes connection separation but does not invent an audit schema.

## Local Docker Middleware

The following persistent containers have been created on Docker network
`easy-admin-dev` and are bound only to `127.0.0.1`:

| Container | Image | Host endpoint | Volume |
| --- | --- | --- | --- |
| `easy-admin-postgres-main` | `postgres:17-alpine` | `127.0.0.1:55432` | `easy-admin-postgres-main-data` |
| `easy-admin-postgres-log` | `postgres:17-alpine` | `127.0.0.1:55433` | `easy-admin-postgres-log-data` |
| `easy-admin-redis` | `redis:8-alpine` | `127.0.0.1:56379` | `easy-admin-redis-data` |

Local DSNs and container names are in the ignored file
`configs/infrastructure.local.env`. Current SRV-001 code can run immediately
with the ignored `configs/config.local.toml`.

Useful commands:

```sh
docker start easy-admin-postgres-main easy-admin-postgres-log easy-admin-redis
docker stop easy-admin-postgres-main easy-admin-postgres-log easy-admin-redis
docker inspect --format '{{.Name}} {{.State.Health.Status}}' \
  easy-admin-postgres-main easy-admin-postgres-log easy-admin-redis
```

Stopping containers preserves data in named volumes. Removing containers or
volumes is not part of normal project operation.

## Configuration Target

SRV-002 replaces the single `[postgres]` section with explicit stores:

```toml
[database.primary]
dsn = "replace-with-secret"

[database.log]
dsn = "replace-with-secret"

[redis]
addr = "127.0.0.1:56379"
password = ""
db = 0
```

Pool and timeout fields remain explicit for each client. Deployment secrets use
named environment overrides; committed examples contain placeholders only.

## Primary PostgreSQL Ownership (USR-001)

Ordered migrations in `migrations/primary/`:

| Migration | Objects | Notes |
| --- | --- | --- |
| `000001_admin_rbac` | `administrators`, `roles`, `permissions`, `administrator_roles`, `role_permissions` | Administrator RBAC only |
| `000002_admin_auth_epoch` | `administrators.auth_epoch` | Authoritative administrator session invalidation |
| `000003_user_levels` | `user_levels` | Seeds the enabled `default` level at threshold `0.0000`; a partial unique index keeps enabled thresholds unambiguous |
| `000004_users` | `users` | Business users. Case-insensitive unique indexes on normalized username (no `@`) and email; `NUMERIC(20,4)` point columns with non-negative checks; `status` and `level_mode` checks; `auth_epoch` |
| `000005_user_point_transactions` | `user_point_transactions` | Immutable ledger: unique idempotency key, both resulting balances, non-negative balance and non-negative consumption checks |
| `000006_system_settings` | `system_settings` | Typed singleton (`id = 1`) with optimistic-lock `version`; seeds safe development defaults |
| `000007_user_rbac_permissions` | `permissions`, `role_permissions` rows | Business-user management permissions; `super_admin` gets all, `admin` gets the user-management surface plus level/settings reads, `finance` gets nothing |

`users`, `user_levels`, `user_point_transactions`, and `system_settings` are
business data and never live in the log database. `smtp_password_encrypted`
holds only AES-256-GCM ciphertext with a random nonce; the 32-byte master key
comes from `SMTP_MASTER_KEY` (or `user.smtp_master_key`) and is never stored in
PostgreSQL or Git. No API, audit detail, runtime log, or CLI output returns the
plaintext or the ciphertext.

## Log PostgreSQL Ownership

`migrations/log/000001_audit_events` creates `audit_events` only. Business-user
and settings mutations follow the same pending-first protocol as administrator
mutations: the `pending` insert happens in the log database before the primary
write, and a failed pending insert aborts the primary write. There is no
cross-database transaction.

## Redis Namespaces

Every key has one documented owner, TTL, invalidation rule, and failure policy.
Administrator and business-user state never share a namespace, cookie, or JWT
audience.

| Key | Owner | Purpose | TTL | Invalidation / failure policy |
| --- | --- | --- | --- | --- |
| `easy-admin:admin-session:<sid>` | SRV-003 | Administrator session (admin id, refresh verifier hash, status, `auth_epoch`) | capped to the session's absolute refresh expiry | logout/replay mark revoked; password and account transitions revoke; missing, revoked, corrupt, expired, or Redis-down fails closed |
| `easy-admin:admin-session-index:<admin_id>` | SRV-003 | SET of that administrator's session ids (best-effort hygiene, never authoritative) | same TTL, refreshed on rotation | deleted after full revocation |
| `easy-admin:user-session:<sid>` | USR-001 | Business-user session document (user id, current refresh verifier hash, status, user `auth_epoch`, absolute expiry) | capped to the absolute refresh expiry (30 days) | rotation is atomic (WATCH + MULTI); reuse of a consumed verifier revokes the session; disable/password reset bump `users.auth_epoch`; any Redis error fails closed |
| `easy-admin:user-session-index:<user_id>` | USR-001 | SET of that user's session ids for best-effort bulk revocation | same TTL, refreshed on rotation | deleted after successful revocation; never authoritative (the `auth_epoch` column is) |
| `easy-admin:user-verify:<user_id>` | USR-001 | SHA-256 of the single active one-time email verification token | `user.verification_token_ttl`, default and maximum 24h | consumed by an atomic Lua GET+DEL compare (one-time, replay-safe); resend overwrites (invalidates the previous token); missing/expired/wrong hash returns the same generic invalid-token error; Redis-down fails closed |
| `easy-admin:rl:register:<ip>` | USR-001 | Fixed-window registration throttle (5/hour/IP) | window length | limiter storage failure is treated as "limited" (fail closed) |
| `easy-admin:rl:login-ip:<ip>`, `easy-admin:rl:login-id:<sha256(identifier)>` | USR-001 | Fixed-window login throttle (10/15min per IP and per identifier) | window length | same fail-closed policy; identifiers are hashed so keys never carry raw account data |
| `easy-admin:rl:resend:<ip>` | USR-001 | Verification resend throttle (3/hour/IP) | window length | same fail-closed policy |
| `easy-admin:rl:verify:<ip>` | USR-001 | Verification redeem throttle (10/hour/IP) | window length | same fail-closed policy |

Integration tests use a dedicated Redis database index and delete only the keys
they created. Test runs create uniquely named disposable primary/log databases
and never fall back to the development databases, truncate shared tables, or
touch Docker volumes.
