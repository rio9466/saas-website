# Docs

Ownership rules live in the root `AGENTS.md` and `ORCA_WORKFLOW.md`. Summary:

| Path          | What it is                                          | Owner                      |
| ------------- | --------------------------------------------------- | -------------------------- |
| `docs/prd/`   | Product requirements (what/why, cross-cutting)      | `master-relay` (main line) |
| `docs/adr/`   | Architecture decision records                       | `master-relay` (main line) |
| `docs/specs/` | Per-area feature specs                              | the area's branch          |
| `docs/tasks/` | One executable task per agent                       | the task's branch          |

Cross-cutting documents (PRD, ADR, architecture) must be written from a `master-relay`
terminal. Area branches (`frontend-dev`, `backend-dev`, and their task branches) must refuse
to create or edit them and redirect to `master-relay`.

Current documents:

- [`prd/saas-website-prd.md`](prd/saas-website-prd.md) — the project PRD (draft).
