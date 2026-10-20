---
name: commit-and-push
description: Save the finished work in git and push it, the last job before the team reports to the person.
always: true
---

# Commit and push

When the orchestrator asks you to save the work, it is done and checked, and the person is about to be told. You do it
with one command, `ship`, so it is always done the same way. Nobody else on the team touches git.

## Do it
1. Write the commit message for the person, who may not be a programmer: a first line under 72 characters that says what
   the work now does ("Add the login page and the member list"), a blank line, then a few short lines on what changed
   and why. No file lists, no jargon.
2. Run it as the first line of one `run_command`: `ship "Add the login page and the member list"`. For a body, keep
   the message in the same quotes across lines. Do not add anything before `ship` on the first line.
3. Read the last lines it prints and use them in your report:
   - `commit: <hash> on dev` is the work saved on the branch dev (made when it was missing; never main).
   - `pushed: yes` means it reached GitHub. `pushed: no (<reason>)` means it did not: no remote or no GitHub yet, no
     permission, or the remote has newer work. Say which, in plain words. Never retry with force, never switch to
     another branch, and never edit git config to get around a refusal.
   - `left out (looks like a secret): <file>` means a password or key was kept out of git. Tell the orchestrator.

## Report
Give the commit hash, the branch, and whether it was pushed with the reason when it was not. A push that failed does
not make the job a failure: the work is saved here. If `ship` stops with an error (exit 3 or 4), report
`Stuck: <the exact message>`.
