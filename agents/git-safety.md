# Git worktree and dirty-state safety

Use the single main checkout at `/workspace` on `dev`. A task must not create
linked, nested, detached, temporary, or staging worktrees, including a
`.stageN/` directory or another checkout folder. Do not move work into a
separate checkout to avoid the preflight.

## Start of every task

Before editing, inspect all of the following without changing repository
state:

1. `git worktree list --porcelain`
2. `git branch --show-current`
3. `git status --short`
4. `git fetch origin`
5. The ahead/behind comparison for `dev` and `origin/dev`.

If Git is unavailable or the folder is not a repository, report that fact and
do not pretend the preflight passed. A preflight is not permission to alter
anything it reveals.

## Stop for a decision

Report and wait for the user when there are pre-existing modifications,
untracked user files, conflicts, an unexpected branch, multiple worktrees,
or local/remote divergence. Ask whether to keep, separate, or discard the
identified state. Never infer that dirty files are disposable or that an
earlier approval covers a new task.

An explicit keep decision applies only to the named starting state and task.
After it, edit only the task's authorized files. Keep new task changes
identifiable from earlier changes, and never stage or commit unrelated files.
Approval to keep is not approval to stash, reset, clean, overwrite, or switch
branches.

## Safe Git operations

- Keep `dev` as the working branch. Do not automatically stash, checkout, or
  switch branches to make a task convenient.
- Reconcile `origin/dev` only with non-destructive operations after assessing
  the ahead/behind state and preserving local work. Do not pull or merge when
  local changes, conflicts, or divergence make the result unclear; report it
  for a decision.
- Never run `git reset --hard`, `git clean -fd` or `git clean -fdx`, `rm -rf`
  on a worktree, `git worktree remove --force`, or a force push.
- Keep `backup/stage7` at commit `ebc4751`. The user must explicitly approve
  deleting that branch.
- Do not commit, push, or deploy unless the task explicitly includes that
  delivery step and its preconditions are satisfied.

The shell guardrail rejects more than one registered worktree and any root
`.stage*/` directory when Git is available in a repository. The manual
preflight remains necessary because the guardrail does not assess branch,
dirty state, conflicts, or remote divergence.
