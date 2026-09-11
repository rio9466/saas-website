# When coding, follow these guidelines:

- Think before coding. State assumptions when the task is ambiguous, and ask only if a reasonable assumption would be risky.
- Prefer the simplest implementation that fully solves the request. Do not add speculative features, abstractions, configurability, or broad error handling unless needed.
- Make surgical changes. Touch only files and lines directly related to the request. Match the existing code style. Do not refactor unrelated code.
- If a change creates unused imports, variables, functions, or files, clean up only those introduced by the change.
- Define verifiable success criteria for non-trivial work. Prefer reproducing bugs with tests, then fixing them. Run relevant tests or explain why they could not be run.
- Surface tradeoffs and uncertainty clearly. Do not hide confusion or silently pick among materially different interpretations.

# Git workflow

- `master-relay` is the AI integration branch and the default base for all AI work. Do not commit directly to `master`.
- Never merge into `master` unless the user explicitly agrees first.
- All branch merges happen on `master-relay`. Branch feature work off `master-relay` and merge back into `master-relay`.
- Long-lived development branches, both based on `master-relay`:
  - `frontend-dev` — frontend development.
  - `backend-dev` — backend development, including the backend admin system.
- Use Orca-managed git worktrees. All development branches (`master-relay`, `frontend-dev`, `backend-dev`) are checked out under `~/orca/workspaces/saas-website/`; the repo root `~/orca/projects/saas-website` stays on `master`.
