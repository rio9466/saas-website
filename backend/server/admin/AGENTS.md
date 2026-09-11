# Admin (server/admin) Instructions

`server/admin/` is the easy-admin management frontend, initialized from
`rio9466/pure-admin-thin` v6.2.0 (see `SOURCE-ADAPTATION.md`). It is outside the
Go service scope; never mix it into the Go module.

## Non-Negotiable Rules

- **Types**: `types/api.generated.ts` is generated from `server/docs/openapi.yaml`
  via `pnpm generate:api`. Never edit it manually; add handwritten adapters in
  `src/api/contract.ts`.
- **Tokens**: access token lives in memory only. Never persist access/refresh
  tokens in localStorage, sessionStorage, IndexedDB, Pinia persistence, or
  script-readable cookies. Refresh token is HttpOnly-cookie-only.
- **Auth flow**: one refresh at a time (single-flight), original request retried
  at most once, refresh failure clears auth state and redirects to login; never
  refresh-loop on login/refresh/401.
- **Routes/menus**: local only, filtered by `/me` role/permission codes. Never
  request routes/menu trees from the backend; direct forbidden navigation shows
  403.
- **UI**: reuse Element Plus and template components; do not add a second UI
  framework. No mock APIs, no demo accounts, no promotional/branding content,
  no fabricated metrics.
- **Backend**: never modify `server/` Go code or `server/docs/openapi.yaml`
  from the admin side; contract issues are reported as blockers.

## Verification

```sh
pnpm install --frozen-lockfile
pnpm generate:api
pnpm lint
pnpm typecheck
pnpm test
pnpm build
```

Browser checks: unauthenticated, login failure/success, 403/404, refresh
restore, logout, mobile width, network requests, cookie flags, and browser
storage must not contain tokens.
