# When coding, follow these guidelines:

- Think before coding. State assumptions when the task is ambiguous, and ask only if a reasonable assumption would be risky.
- Prefer the simplest implementation that fully solves the request. Do not add speculative features, abstractions, configurability, or broad error handling unless needed.
- Make surgical changes. Touch only files and lines directly related to the request. Match the existing code style. Do not refactor unrelated code.
- If a change creates unused imports, variables, functions, or files, clean up only those introduced by the change.
- Define verifiable success criteria for non-trivial work. Prefer reproducing bugs with tests, then fixing them. Run relevant tests or explain why they could not be run.
- Surface tradeoffs and uncertainty clearly. Do not hide confusion or silently pick among materially different interpretations.

# Document ownership (strong rules)

- Cross-cutting documents are owned by the main line (`master-relay`, later published to `master`): the PRD (`docs/prd/`), ADRs (`docs/adr/`), architecture, this root `AGENTS.md`, and `ORCA_WORKFLOW.md`.
- If you are on `frontend-dev`, `backend-dev`, or any area/feature task branch, do NOT create or edit those documents. Stop and tell the user: "This is cross-cutting documentation owned by master-relay; open a pi terminal on master-relay (or a docs branch cut from it)."
- An area branch owns only its own area's specs and `AGENTS.md` (`frontend/AGENTS.md` or `backend/AGENTS.md`). Never write the other area's docs.
- The orchestrator writes task documents on `master-relay`; an executor reads its assigned task doc and must not rewrite it without approval.
- Refuse these requests even when asked to do them "just this once". Redirect to the correct branch instead.

# Git workflow

- `master-relay` is the AI integration branch and the default base for all AI work. Do not commit directly to `master`.
- Never merge into `master` unless the user explicitly agrees first.
- All branch merges happen on `master-relay`. Branch feature work off `master-relay` and merge back into `master-relay`.
- Long-lived development branches, both based on `master-relay`:
  - `frontend-dev` — frontend development.
  - `backend-dev` — backend development, including the backend admin system.
- Use Orca-managed git worktrees. All development branches (`master-relay`, `frontend-dev`, `backend-dev`) are checked out under `~/orca/workspaces/saas-website/`; the repo root `~/orca/projects/saas-website` stays on `master`.

### Branch constraints

- Each branch edits only its own area: frontend branches edit `frontend/`, backend branches edit `backend/`. Cross-area changes need separate task branches or an explicit exception in the task document.
- New task branches are cut from `master-relay` and merged back into `master-relay`; `master` stays untouched unless the user explicitly approves.
- Never reintroduce easy-admin's git history, add it as a submodule, or re-add the original repo as a remote.
- Area rules live in `frontend/AGENTS.md` and `backend/AGENTS.md`; the nearest layered `AGENTS.md` wins.
- Full workflow, roles, task dispatch, and initialization paths: see `ORCA_WORKFLOW.md`.
