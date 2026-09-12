<!-- BEGIN:nextjs-agent-rules -->

# This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` (resolved from this file's directory; in monorepos the `next` package may not be visible from the repo root) before writing any code. Heed deprecation notices.

This block is written and re-added by `next dev` — verify at `node_modules/next/dist/server/lib/generate-agent-files.js`. Removing it from a diff only re-creates the uncommitted change; committing it with your work keeps the tree clean.

<!-- END:nextjs-agent-rules -->

# Next template (`next/`) Instructions

Owned by the `next` task branches cut from `master-relay`. No other area
(including `frontend/` and `backend/`) may modify this directory, and this
branch must not modify `frontend/` or `backend/`.

## Stack

- Next.js 16 (App Router / RSC) + React 19 + TypeScript + Tailwind CSS v4 +
  shadcn/ui (base-nova style, `@base-ui/react`) + `next-intl` + `next-themes` +
  `lucide-react`.
- This is an **independent pnpm project** (not a pnpm workspace member of
  `frontend/`).
- Dev/start port is **3200**; it is fixed in the `dev`/`start` scripts.

## Rules

- Reuse the shared API contract `docs/api/frontend-api-contract.md`; never
  change request/response shapes. If the contract is missing something, stop
  and report to `master-relay`.
- The browser only calls the same-origin `/api/**` route handler
  (`app/api/[...path]/route.ts`), which forwards to `API_PROXY_TARGET` at
  runtime. Never add build-time `next.config` rewrites and never call the
  backend origin from the browser. SSR uses `API_INTERNAL_BASE`.
- All API calls go through `lib/api.ts` (envelope unwrap, `ApiError`,
  Bearer from memory, `X-Request-ID`, single 401 refresh retry). Never persist
  tokens to `localStorage`/`sessionStorage`/cookies.
- Use `Link`/`redirect`/`useRouter` from `@/i18n/navigation` for in-site
  links; a bare `/x` drops the locale prefix. Locale routing lives in
  `i18n/routing.ts` (`en` unprefixed, `zh-CN` under `/zh-CN`), persisted via
  the `NEXT_LOCALE` cookie.
- Next.js 16 renamed Middleware to Proxy: locale routing lives in `proxy.ts`.
- Keep the scaffold's existing style; do not reformat unrelated files.

## Verification (before reporting done)

```bash
cd next
pnpm lint && pnpm build
API_PROXY_TARGET=http://127.0.0.1:8100 API_INTERNAL_BASE=http://127.0.0.1:8100 pnpm dev -p 3200
```

## Refusals (strong rules)

- Must not create or edit cross-cutting documents: the PRD (`docs/prd/`),
  ADRs, architecture, root `AGENTS.md`, `ORCA_WORKFLOW.md`, `docs/tasks/**`,
  or the API contract. Refuse and tell the user to open a pi terminal on
  `master-relay`.
- Never create or edit `frontend/` or `backend/`.

## See also

- Root `AGENTS.md` for coding guidelines and git rules, and `ORCA_WORKFLOW.md`
  for the branch model.
