# UI PRD v2 — QA Completion / Remaining Blockers (2026-10-30)
**Status:** PARTIALLY VERIFIED — NOT 100% COMPLETE. **Target:** agentic-ai-workspace/dev at c39a94d (baseline), implementation already shipped in 2e6a40f. This document covers the post-implementation closure attempt.

## Verified in an isolated application (no external deployment changes)
- Frontend: `npm --prefix web run verify` passed Svelte check (0 errors/warnings), ESLint and Vite build; shell syntax and `git diff --check` passed.
- Visual routes: `scripts/shots.sh` verified document-level horizontal overflow on **30 route/tab configurations at widths 390, 820, 1024 and 1440px = 120 checks/captures**. The actual test condition is `scrollWidth <= innerWidth+1`; screenshot presence alone is not treated as proof of contrast/accessibility.
- Found/fixed `SettingsSection.svelte` header width overflow on Settings > Team at 820px (document width 837px); allow action/header to wrap. Recheck all four widths passed.
- Browser `e2e projects`: wizard handles GitHub unavailable/broken/disconnected/ready, repo picker search, import existing repo, create/import new repository; Checks CRUD and Escape dismissal; project Checks no horizontal overflow at 4 widths; **all passed**.
- Browser `e2e git`: changed file and parent folder badges, editor comparison gutter, live diff; **all passed**. Test fixture now changes tracked files AFTER task bootstrap because bootstrapping reset earlier fixture edits.
- Browser `e2e lifecycle`: reported/done/reconnect, idle stop, plan ready-for-approval, four-width approval task overflow and Plan/Documents/Team/Changes/Files tab access; **all passed**.
- Browser `e2e context`: measured context, compacting and failed/unknown display; **all passed**.
- Black-box `07-external-work-lifecycle`: **58/58 passed** after updating stale expectations to the existing `auto_completed_unverified` Done contract (backend unchanged).
- Black-box `10-mcp-research-plan-workflow`: **23/23 passed**. Earlier Git API status/metadata black-box checks passed 228 assertions.
- No authorization or task lifecycle backend logic altered by this closure attempt.

## Actual runtime blockers — cannot certify 100%
1. **Docker-enabled host absent in this coding sandbox:** no `docker` executable and no `/var/run/docker.sock`. Full `e2e thread` ran through illustrated question and saved answer, then waited 120s for the model tool workflow to reach Result; sandbox site/start + screenshot stage cannot be verified. `e2e sites` also needs a real container/site runner. These are not marked passed.
2. **No model provider configured for screenshots:** throwaway screenshot app returns `HTTP 409 no_model` when creating tasks. Populated model-driven chats and error/permission variations cannot all be captured from that fixture.
3. Full contrast/WCAG AA, reduced-motion and all permission/disabled/error/streaming edge states are NOT independently proven across 154 components; static 154/154 inventory is NOT runtime test coverage.

## Remaining acceptance / unblock
- Run `bash test/run.sh e2e thread` and `bash test/run.sh e2e sites` on a Docker-enabled host with a working sandbox image, confirm final Result, screenshot cards, protected site flow and streaming.
- With configured model or isolated scripted model fixtures, capture filled/long/failed task states, question/approval/review states, empty/loading/errors, admin vs user, and modal focus/keyboard; inspect actual screenshots at 390/820/1024/1440.
- Run focused accessibility/contrast and reduced-motion verification, then compare before/after and explicitly sign off remaining PRD checks. Do not declare the project 100% complete until those gates pass.
- Existing implemented visual system remains preserved; only the proven Teams responsive fix and test harness repairs were changed in this pass.
