# saas-website

Monorepo for the SaaS website: a Nuxt frontend and an easy-admin-based backend.

## Structure

| Path        | Stack                                                                                     | Notes                                                                             |
| ----------- | ----------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| `frontend/` | Nuxt 4 + Nuxt UI + Tailwind CSS, managed with pnpm                                         | Public-facing site. Not wired to the backend yet.                                 |
| `backend/`  | easy-admin: Go API (`server/`) + Vue/Vite admin console (`server/admin/`)                   | Vendored from https://github.com/rio9466/easy-admin; upstream git history removed. |

## Requirements

- Node.js 22+ and pnpm. `frontend/` pins its pnpm version via `packageManager`.
- Go toolchain, PostgreSQL, and Redis for the backend server.

## Frontend

```bash
cd frontend
pnpm install
pnpm dev     # http://localhost:3000
pnpm build   # production build
```

## Backend

See `backend/server/README.md` for the Go API, `backend/server/admin/README.md` for the
admin console, and `backend/server/docs/` for architecture and API contract details.

## Development workflow (Orca)

- `master-relay` is the AI integration branch. Do not commit directly to `master`, and do
  not merge into `master` without explicit user approval.
- Work happens in Orca-managed git worktrees under `~/orca/workspaces/saas-website/<branch>`;
  the repo root (`~/orca/projects/saas-website`) stays on `master`.
- Long-lived branches: `master-relay` (integration), `frontend-dev`, `backend-dev`.
- To start a task: switch to the target branch in Orca, open a pi terminal in that worktree,
  and hand pi the task document. Feature work merges back into `master-relay`.
- Full workflow reference (roles, task dispatch, initialization paths): `ORCA_WORKFLOW.md`.
