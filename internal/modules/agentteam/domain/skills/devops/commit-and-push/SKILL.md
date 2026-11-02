---
name: commit-and-push
description: Save the finished work in git and push it, the last job before the team reports to the person.
always: true
---

# Commit and push

Use the native `git` tool first for ordinary Git work, including status, log, diff, fetch, add, commit, push and
branch operations allowed by the repository's Git safety rules. Do not use shell commands or `ship` as a substitute
for ordinary Git operations. `ship` is for an actual build or release workflow, not routine Git work.

## Do it
1. Inspect the branch and worktree with the native Git tool. Follow repository git-safety rules, preserve unrelated
   changes, and stage only files authorized for this task.
2. When the orchestrator asks you to save the finished work, write a plain commit message: a first line under 72
   characters that says what changed, a blank line, then a few short lines on what changed and why. No file lists or
   jargon.
3. Use the native Git tool to stage, commit and push the authorized changes. Never force push, switch branches, or
   change Git configuration to get around a refusal. Report any write or remote error as a blocker; do not claim a
   commit or push succeeded without its result.
4. Use `ship` only when an actual build or release workflow requires it, not for ordinary Git operations.

## Report
Give the commit hash, the branch, and whether it was pushed, with the reason when it was not. Never imply work was
saved or published when Git reports a failure.
