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

## 4. Initialization scenarios

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

## 5. Branch constraints (AGENTS.md)

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

## 6. Merge and review

1. The executor runs the acceptance checks in the task doc and reports the exact command and
   result.
2. Merge `--no-ff` into `master-relay`; do not fast-forward away the task history.
3. After a merge, `master-relay` must still build/validate for the areas it touched.
4. `master` is updated only with explicit user approval.

## 7. Environment notes

- **pnpm 12 on macOS** — the version manager can install pnpm without its native binary,
  leaving a shebang-less `bin/pnpm` that fails with `ENOEXEC`. Fix:
  `node ~/Library/pnpm/.tools/pnpm/<version>/node_modules/pnpm/install.js`.
- **raw.githubusercontent.com** can fail transiently during `create-nuxt`; retry. Cloning
  from `github.com` is a reliable fallback.
- Orca injects pi extensions for status reporting, the terminal-title spinner, and editor
  prefill. They are managed by Orca; do not edit them by hand.

## 8. Task document template

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
