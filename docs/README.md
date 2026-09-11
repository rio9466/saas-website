# Docs

Ownership rules live in the root `AGENTS.md` and `ORCA_WORKFLOW.md`. Summary:

| Path          | What it is                                          | Owner                      |
| ------------- | --------------------------------------------------- | -------------------------- |
| `docs/prd/`   | Product requirements (what/why, cross-cutting)      | `master-relay` (main line) |
| `docs/api/`   | Frontend ↔ backend API contract (cross-cutting)     | `master-relay` (main line) |
| `docs/adr/`   | Architecture decision records                       | `master-relay` (main line) |
| `docs/specs/` | Per-area feature specs                              | the area's branch          |
| `docs/tasks/` | One executable task per agent                       | the task's branch          |

Cross-cutting documents (PRD, ADR, architecture) must be written from a `master-relay`
terminal. Task branches must refuse
to create or edit them and redirect to `master-relay`.

Current documents:

- [`prd/saas-website-prd.md`](prd/saas-website-prd.md) — the project PRD (v0.1, decomposed into tasks).
- [`api/frontend-api-contract.md`](api/frontend-api-contract.md) — the frontend ↔ backend API contract
  (envelope, auth, error codes, endpoint shapes, frontend rules).
- [`tasks/README.md`](tasks/README.md) — task index and execution order (BE-01…04, FE-01…03).
