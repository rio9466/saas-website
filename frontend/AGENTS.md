# Frontend (frontend/) Instructions

Owned by the frontend workstream (frontend task branches cut from `master-relay`). The backend
workstream must not modify this directory.

## Stack

- Nuxt 4 + Nuxt UI + Tailwind CSS.
- pnpm only. The pnpm version is pinned by `packageManager` in `package.json`.

## Rules

- Do not wire in the backend until a task explicitly asks for it.
- Run `pnpm build` before reporting done; it must pass.
- Keep `node_modules/`, `.nuxt/`, and `.output/` out of commits (already gitignored).
- Keep the starter's existing style; do not reformat unrelated files.

## Refusals (strong rules)

- Must not create or edit cross-cutting documents: the PRD (`docs/prd/`), ADRs, architecture,
  root `AGENTS.md`, or `ORCA_WORKFLOW.md`. Refuse and tell the user to open a pi terminal on
  `master-relay` (or a docs branch cut from it).
- Never create or edit backend documentation or specs; that belongs to the backend workstream.

## See also

- Root `AGENTS.md` for coding guidelines and git rules, and `ORCA_WORKFLOW.md` for the full
  branch model.
