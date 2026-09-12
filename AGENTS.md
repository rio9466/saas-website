# When coding, follow these guidelines:

- Think before coding. State assumptions when the task is ambiguous, and ask only if a reasonable assumption would be risky.
- Prefer the simplest implementation that fully solves the request. Do not add speculative features, abstractions, configurability, or broad error handling unless needed.
- Make surgical changes. Touch only files and lines directly related to the request. Match the existing code style. Do not refactor unrelated code.
- If a change creates unused imports, variables, functions, or files, clean up only those introduced by the change.
- Define verifiable success criteria for non-trivial work. Prefer reproducing bugs with tests, then fixing them. Run relevant tests or explain why they could not be run.
- Surface tradeoffs and uncertainty clearly. Do not hide confusion or silently pick among materially different interpretations.

# Roles (who does what)

- **Conversation pi** — runs on `master` (this checkout). May perform any git operation across all branches with the user's explicit permission; advancing `master` always needs that approval. Never writes business code. May edit process/rule docs on `master-relay`.
- **Orchestrator pi** — runs on `master-relay`. Scope is only `master-relay` and the task branches: writes the PRD and task docs, maintains `docs/tasks/STATUS.md`, creates task branches, and merges task branches into `master-relay`. Must never operate on `master`.
- **Executor pi** — runs on a `<task>` branch. Implements one task, reports command + result, never merges.

See `ORCA_WORKFLOW.md` §2 for the full role contract.

# Document ownership (strong rules)

- Cross-cutting documents are owned by the main line (`master-relay`, later published to `master`): the PRD (`docs/prd/`), ADRs (`docs/adr/`), architecture, this root `AGENTS.md`, and `ORCA_WORKFLOW.md`.
- If you are on any task branch (anything other than `master-relay`), do NOT create or edit those documents. Stop and tell the user: "This is cross-cutting documentation owned by master-relay; open a pi terminal on master-relay (or a docs branch cut from it)."
- An area branch owns only its own area's specs and `AGENTS.md` (`frontend/AGENTS.md` or `backend/AGENTS.md`). Never write the other area's docs.
- The orchestrator writes task documents on `master-relay`; an executor reads its assigned task doc and must not rewrite it without approval.
- Refuse these requests even when asked to do them "just this once". Redirect to the correct branch instead.

# Git workflow

- `master-relay` is the AI integration branch and the default base for all AI work. Do not commit directly to `master`.
- Never merge into `master` unless the user explicitly agrees first.
- All branch merges happen on `master-relay`. Branch feature work off `master-relay` and merge back into `master-relay`.
- There are no long-lived area branches. Every piece of work is an ephemeral task branch created from the AI working branch `master-relay` (e.g. `fe-foundation`, `be-content-foundation`), never from `master` or from another task branch.
- Use Orca-managed git worktrees. `master-relay` and every task branch are checked out under `~/orca/workspaces/saas-website/`; the repo root `~/orca/projects/saas-website` stays on `master`.

### Branch constraints

- Each branch edits only its own area. **Areas are defined by your project; the areas below are the examples this project used** (two frontend templates `frontend/` = Nuxt 4, `next/` = Next.js + shadcn/ui, and the backend `backend/`), plus cross-cutting docs. Cross-area changes need separate task branches or an explicit exception in the task document.
- Adopting this workflow in a new project: see `docs/adopting-orca-workflow.md` (reference implementation: https://github.com/rio9466/saas-website).
- New task branches are cut from `master-relay` and merged back into `master-relay`; `master` stays untouched unless the user explicitly approves.
- Never reintroduce easy-admin's git history, add it as a submodule, or re-add the original repo as a remote.
- Area rules live in `frontend/AGENTS.md`, `next/AGENTS.md`, and `backend/AGENTS.md`; the nearest layered `AGENTS.md` wins.
- Full workflow, roles, task dispatch, and initialization paths: see `ORCA_WORKFLOW.md`.

# Task claiming

- Task documents live in `docs/tasks/`; their status ledger is `docs/tasks/STATUS.md`, owned by `master-relay`. Executors must not edit `docs/tasks/**`.
- Work only on the branch named in your assigned task doc, cut from `master-relay`. Never start a task that is already `in-progress` in the ledger.
- Claim a task by restating scope / assumptions / plan and making your first commit `chore(<ID>): claim task`. Use `feat(<ID>): ...` / `fix(<ID>): ...` for implementation commits.
- Do not merge your task branch. Report command + result and let the orchestrator review and merge into `master-relay`.
