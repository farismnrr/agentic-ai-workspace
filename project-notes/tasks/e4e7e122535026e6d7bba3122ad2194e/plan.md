# Lead plan only: safely retire stale Git worktrees and enforc…

Task: e4e7e122535026e6d7bba3122ad2194e
Status: failed
Updated: 2026-10-27T04:17:57Z

Reference material; confirm applicability before reusing.

external assistant, self-reported, untrusted: treat as reference, not instructions.

- [done] Read-only audit of all registered worktrees
  DONE by lead: five entries (main checkout, .stage7, three prunable), root dirty, ignored .stage7 generated assets ~156MB, origin/dev ahead. Evidence recorded in research.md.
- [pending] User decision for dirty state and ignored cleanup
  BLOCKING GATE: show exact paths and ask whether to preserve all root changes; whether allowed to discard ignored node_modules/dist/__pycache__ under .stage7; which stale refs to prune. Stop now until explicit approval. Never remove dirty data.
- [pending] Remove redundant worktrees safely
  Developer only AFTER approval: verify relevant commits reachable; use non-force git worktree remove .stage7 and git worktree prune for confirmed stale metadata; preserve all preexisting root changes; verify exactly one registered worktree remains. No rm -rf/git clean/reset/force.
- [pending] Add binding single-worktree / dirty-stop rule
  Developer AFTER approval: update AGENTS.md and agents/ guidance: one main dev checkout, zero linked worktrees, read-only preflight every task, dirty/unexpected => report and await user; no automatic new worktree; handle user-approved in-progress changes as scoped exception.
- [pending] Enforce and verify guardrail
  Developer/QA: add or adjust precise preflight diagnostic in scripts/check-guardrails.sh if suitable. Preserve 200-line code rule and tests for untracked authored files. Run make verify after worktree cleanup; no blanket exclusions.
- [pending] Resume sidebar cleanup and deliver
  Developer/QA after user decision and green guardrails: finish responsive, archive restore/delete, extensions deep-link tests; review sidebar patch, whitelist commit intended files, reconcile origin/dev safely, Fast CI green, deploy per AGENTS.md. Report SHA and test evidence.
