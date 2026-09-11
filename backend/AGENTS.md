# Backend (backend/) Instructions

Owned by the backend workstream (`backend-dev` and backend task branches). The frontend
workstream must not modify this directory.

## Structure

- `server/` — Go API. Its rules live in `server/AGENTS.md`.
- `server/admin/` — easy-admin management console. Its rules live in `server/admin/AGENTS.md`.

## Origin

Vendored from https://github.com/rio9466/easy-admin as plain source.

- Never reintroduce its `.git`, add it as a submodule, or re-add the original repo as a
  remote.
- The original upstream repo is not part of this project; keep it untouched.

## Rules

- Respect the rule files in `server/` and `server/admin/`.
- Keep `server/` (Go) and `server/admin/` (console) separate; do not mix them into one
  module or build.
- `server/docs/` is tracked even though the local `docs/` ignore pattern would hide it; keep
  it in sync with the OpenAPI contract when changing the API.
- See the root `AGENTS.md` for coding guidelines and git workflow, and `ORCA_WORKFLOW.md`
  for the full branch model.
