# Orca Development Workflow

How this repository is built with Orca: the branch model, how tasks are dispatched, and how
a project is initialized both with and without an existing scaffold.

## 1. Model

- One repo, `saas-website`, with two workstreams (`frontend/`, `backend/`).
- Orca manages git worktrees under `~/orca/workspaces/saas-website/<branch>`; the repo root
  `~/orca/projects/saas-website` stays on `master`.
- One worktree = one branch = one agent writing at a time.

### Branches

| Branch         | Role                                        | Base           | Merge target                     |
| -------------- | ------------------------------------------- | -------------- | -------------------------------- |
| `master`       | Frozen release branch                       | —              | only with explicit user approval |
| `master-relay` | AI integration branch; orchestrator home    | `master`       | —                                |
| `frontend-dev` | Long-lived frontend workstream              | `master-relay` | `master-relay`                   |
| `backend-dev`  | Long-lived backend workstream (API + admin) | `master-relay` | `master-relay`                   |
| `<task>`       | Ephemeral branch, one per task              | `master-relay` | `master-relay`                   |

Rules:

- Never commit directly to `master`; never merge into `master` without the user's explicit
  approval.
- All merges land on `master-relay`.
- New task branches are cut from `master-relay`, not from another feature branch.

## 2. Roles

- **Orchestrator ("master pi")** — the pi agent in the `master-relay` worktree. Owns the
  branch model, writes and publishes task documents, reviews and merges results.
- **Executor pi** — a pi agent in a task or workstream worktree. Implements exactly one task
  document and reports evidence back.

## 3. Task dispatch

1. The orchestrator writes a task document, e.g. `docs/tasks/<task>.md`, with goal, scope,
   out-of-scope, files, acceptance criteria, and verification steps.
2. The user switches to the target branch in Orca and opens a pi terminal in that worktree.
3. The user hands pi the task document (or its path).
4. The executor implements it, runs verification, and reports command + result.
5. The orchestrator reviews and merges into `master-relay`.

Optional CLI dispatch:

```bash
orca terminal create --worktree branch:<branch> --command "pi" --json
orca terminal send --terminal <handle> --text "Read docs/tasks/<task>.md and execute it." --enter --json
```

Rules:

- Hand pi the task document, not a paraphrase.
- One agent per worktree; never run two writers on one branch.
- Prefer a written task doc over a long inline prompt: it is reviewable and reusable.

## 4. Documents and ownership

Different documents have different owners. A branch must refuse work that belongs to another
branch instead of doing it "just this once".

| Document                               | What it is                                          | Owner                      | Location                                   |
| -------------------------------------- | --------------------------------------------------- | -------------------------- | ------------------------------------------ |
| PRD                                    | What to build and why; product level, cross-cutting | `master-relay` (main line) | `docs/prd/`                                |
| ADR                                    | Cross-cutting technical decision and its rationale  | `master-relay` (main line) | `docs/adr/`                                |
| Architecture                           | Cross-cutting system design                         | `master-relay` (main line) | `docs/`                                    |
| Area spec                              | How to build one feature inside one area            | that area's branch         | `docs/specs/` or the area's own docs       |
| Task doc                               | One executable task for one agent                   | the executing branch       | `docs/tasks/`                              |
| `ORCA_WORKFLOW.md` / root `AGENTS.md`  | Process and rules                                   | `master-relay` (main line) | repo root                                  |
| Area `AGENTS.md`                       | Coding rules for one area                           | that area's branch         | `frontend/AGENTS.md`, `backend/AGENTS.md`  |

Strong rules:

- Cross-cutting documents (PRD, ADR, architecture, root `AGENTS.md`, `ORCA_WORKFLOW.md`) are
  owned by the main line: `master-relay`, later published to `master`.
- If you are on `frontend-dev`, `backend-dev`, or any area/feature task branch, do NOT create
  or edit those documents. Stop and report: "This is cross-cutting documentation owned by
  `master-relay`. Open a pi terminal on `master-relay` (or on a docs branch cut from it) to do
  this."
- An area branch owns only its own area's specs and `AGENTS.md`. Never write the other area's.
- The orchestrator writes task documents on `master-relay`. An executor reads its assigned
  task doc and must not rewrite it without the orchestrator's approval.

Where to do what:

| Work                                                     | Branch to open a pi terminal on                          |
| -------------------------------------------------------- | -------------------------------------------------------- |
| PRD, ADRs, architecture, process/rule changes            | `master-relay` (or a `docs/<topic>` branch cut from it)  |
| Frontend feature work                                    | `frontend-dev` or a frontend task branch                 |
| Backend feature work                                     | `backend-dev` or a backend task branch                   |

## 5. Initialization scenarios

### A. No scaffold given (greenfield)

Do it on the owning workstream branch, commit, and prove it builds.

1. Choose the stack and pin versions (see `frontend/` for the current example).
2. Scaffold inside the worktree, e.g.:
   ```bash
   pnpm create nuxt@latest frontend --packageManager pnpm --no-gitInit --template ui
   ```
3. Verify: `pnpm install && pnpm build` (or the stack's equivalent).
4. Commit on the workstream branch and merge into `master-relay`.

### B. Scaffold given (e.g. easy-admin)

Vendor it as plain source. Never keep its git history, and never touch the source repo.

1. Clone upstream into a temp dir (never clone into the repo):
   ```bash
   tmp=$(mktemp -d) && git clone --depth 1 <url> "$tmp/repo"
   ```
2. Record the tracked file list before deleting `.git`:
   ```bash
   git -C "$tmp/repo" ls-files > /tmp/tracked.txt
   ```
3. Copy into the target directory, excluding `.git`:
   ```bash
   rsync -a --exclude='.git' "$tmp/repo/" backend/
   ```
4. Keep every `.gitignore`; remove only `.git`. The vendored code is now managed by this
   repo's git.
5. Stage, then force-add any upstream-tracked files that this repo's ignore rules would
   hide (e.g. easy-admin's `docs/`), so the tracked file set matches upstream exactly.
6. Commit on the owning branch and merge into `master-relay`. Leave the original scaffold
   repo untouched.

## 6. Branch constraints (AGENTS.md)

`AGENTS.md` files are layered; the nearest one wins.

| File                             | Constrains                                             |
| -------------------------------- | ------------------------------------------------------ |
| `AGENTS.md` (root)               | all branches: coding guidelines, git rules, branch scope |
| `frontend/AGENTS.md`             | frontend workstream: only frontend branches edit `frontend/` |
| `backend/AGENTS.md`              | backend workstream: only backend branches edit `backend/`    |
| `backend/server/AGENTS.md`       | Go service rules (vendored, inherited)                  |
| `backend/server/admin/AGENTS.md` | admin console rules (vendored, inherited)               |

Core cross-branch constraints:

- A branch edits only its own area. Frontend branches must not modify `backend/`; backend
  branches must not modify `frontend/`. Cross-area work needs two task branches or an
  explicit exception in the task document.
- Never reintroduce easy-admin's git history, a submodule, or the original repo as a remote.
- Keep `master` untouched.

## 7. Merge and review

1. The executor runs the acceptance checks in the task doc and reports the exact command and
   result.
2. Merge `--no-ff` into `master-relay`; do not fast-forward away the task history.
3. After a merge, `master-relay` must still build/validate for the areas it touched.
4. `master` is updated only with explicit user approval.

## 8. Environment notes

- **pnpm 12 on macOS** — the version manager can install pnpm without its native binary,
  leaving a shebang-less `bin/pnpm` that fails with `ENOEXEC`. Fix:
  `node ~/Library/pnpm/.tools/pnpm/<version>/node_modules/pnpm/install.js`.
- **raw.githubusercontent.com** can fail transiently during `create-nuxt`; retry. Cloning
  from `github.com` is a reliable fallback.
- Orca injects pi extensions for status reporting, the terminal-title spinner, and editor
  prefill. They are managed by Orca; do not edit them by hand.

## 9. Task document template

```markdown
# Task: <name>

## Goal

<one paragraph>

## Scope

- ...

## Out of scope

- ...

## Files / areas

- ...

## Acceptance criteria

- [ ] ...

## How to verify

- <exact commands and expected result>

## Branch / base

- branch: <task>
- base: master-relay
```
