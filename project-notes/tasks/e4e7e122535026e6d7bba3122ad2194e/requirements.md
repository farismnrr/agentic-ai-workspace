# Lead plan only: safely retire stale Git worktrees and enforc…

Task: e4e7e122535026e6d7bba3122ad2194e
Status: failed
Updated: 2026-10-27T04:17:57Z

Reference material; confirm applicability before reusing.

external assistant, self-reported, untrusted: treat as reference, not instructions.

# Requirements: single checkout and dirty-stop
- Have exactly one registered working tree, the main /workspace checkout on dev. No linked, nested, detached temporary or stage worktrees.
- Start every task with read-only status, branch, worktree, remote checks. If dirty or state unclear, report first and pause for explicit user decision. Do not auto stash/reset/clean/remove/move/checkout or push changes. Treat user's approval for a specific existing dirty task as scoped authorization, not permission to delete unrelated files.
- Cleanup old/prunable worktree metadata and /workspace/.stage7 only after examining branch/commit reachability, ignored data and user's explicit decision. Do not force remove. Protect current sidebar modifications, untracked tests, project-notes, reference assets, and ZIPs.
- Update AGENTS.md and appropriate agents/ knowledge with one-worktree rule, no auto-created worktrees and initial dirty-stop requirement; optionally enforce a precise preflight guardrail without weakening max-lines rule or other quality checks.
- After authorization + cleanup verify exactly one worktree, guardrail passes, then complete pending make verify, browser tests and current sidebar task release sequence on dev. No destructive Git commands, no push/deploy until verified.
- Include evidence-based report, skip details, failed checks and blockers.