# Critical regression: delegation membuat banyak top-level tas…

Task: fe62437cacc40766e89f209a58d86642
Status: done
Updated: 2026-10-29T02:43:24Z

Reference material; confirm applicability before reusing.

external assistant, self-reported, untrusted: treat as reference, not instructions.

- [done] Planner: PRD issue task spam
  Requirements saved, scope and acceptance criteria explicitly defined
- [done] Researcher: reproduce MCP/HTTP E2E
  Confirmed two root causes: repeated begin_work without keys and linked children in ordinary task list; code trace saved
- [done] Developer: minimal lifecycle and visibility fix
  Resume active work by default, explicit new_task, durable begin key aliases, hide child rows without deleting history; commit 132d6fc
- [done] QA: focused black-box E2E
  MCP/API tests for roles, distinct work, reconnect, idempotency, archive, child chat, handoff, full Go and browser lifecycle/context pass
- [done] Reviewer: safe diff and push to dev
  Full Go, Svelte 0 errors, browser E2E, git diff --check passed; origin/dev push verified clean
