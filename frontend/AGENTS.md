# Frontend (frontend/) Instructions

Owned by the frontend workstream (`frontend-dev` and frontend task branches). The backend
workstream must not modify this directory.

## Stack

- Nuxt 4 + Nuxt UI + Tailwind CSS.
- pnpm only. The pnpm version is pinned by `packageManager` in `package.json`.

## Rules

- Do not wire in the backend until a task explicitly asks for it.
- Run `pnpm build` before reporting done; it must pass.
- Keep `node_modules/`, `.nuxt/`, and `.output/` out of commits (already gitignored).
- Keep the starter's existing style; do not reformat unrelated files.
- See the root `AGENTS.md` for coding guidelines and git workflow, and `ORCA_WORKFLOW.md`
  for the full branch model.
