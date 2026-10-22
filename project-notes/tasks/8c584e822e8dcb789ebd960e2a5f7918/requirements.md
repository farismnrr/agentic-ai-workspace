# sambil parallel revisi dong

Task: 8c584e822e8dcb789ebd960e2a5f7918
Status: done
Updated: 2026-10-21T13:38:46Z

Reference material; confirm applicability before reusing.

**Scope:** Large. It adds a database column, new batch endpoints, a new Archive view, a multi-select bar across three lists, and file cleanup on permanent delete. I have not re-read the code. This plan relies on the research and the Business Analyst's requirements.

**Out of scope:**
- The Postgres sandbox network issue. It is by design, because the sandbox runs with `NetworkMode: none`.
- Auto-delete of old archived tasks.
- Undo after a permanent delete.
- Deleting shared project uploads. Tasks have no ownership link to them.
- Project-level archive, which already exists.

**Done when:**
1. Archiving a task, or several selected at once, removes it from the sidebar (Recent and Pinned), the global Tasks list, the project Tasks tab and the needs-input counts. It then appears in Archive. Restore brings it back and keeps its pin. An archived task page is read-only with a banner and a Restore button.
2. "Delete permanently" asks "Delete N tasks?". After confirming, the task's database rows, screenshots and `project-notes/tasks/<id>/` are gone, the notes index no longer lists them, and shared uploads are untouched. A running task is stopped first, and the dialog warns about that.
3. `make verify` is green, with tests for archive, restore, delete, delete-while-running, a batch of more than 200 tasks, and file cleanup. Fast CI is green on `dev`, and `/api/health` on port 8081 answers after `make redeploy-dev`.

**Todo:**
1. **Developer: archive state in the backend.**
   - Add the `archived_at` migration, the domain field and the store methods.
   - List queries, the needs-input count and the 200 limit exclude archived tasks. A new archived-list query serves the Archive view.
   - Verify: unit tests show an archived task is absent from the normal list and counts, and present in the archived list.
2. **Developer: batch archive, restore and permanent delete endpoints.**
   - The three batch endpoints take many ids and are idempotent. Already-done counts as success, and a partial failure is reported per id.
   - Delete stops running tasks, then removes screenshots and `project-notes/tasks/<id>/`, deletes rows and rebuilds the notes index.
   - The task page needs an archive action and read-only guard.
   - `DeleteAll` now cleans up every task, not just the first 200.
   - Verify: Go tests for archive, restore, delete, delete-while-running, batch over 200, file cleanup and the `DeleteAll` gap.
3. **Developer: Archive view in the web app.**
   - Add a route and sidebar entry. Each row has Restore and Delete permanently, with a confirmation dialog that states the count.
   - Lists exclude archived tasks. Use existing shared UI and daisyUI components.
   - Verify: open Archive in the running app, restore one task and delete one; `make verify` passes.
4. **Developer: multi-select action bar in the lists.**
   - Add checkboxes and "select all shown" with an action bar in the global Tasks list, the project Tasks tab and the Archive view.
   - The bar offers Archive in the first two and Restore or Delete in Archive. Partial failures show in a toast.
   - Verify: select several tasks and archive them in one action.
5. **Developer: archive on the task page.**
   - Add a single Archive action. An archived task page shows a banner with Restore and its inputs are read-only. A pinned task keeps its pin after restore.
   - Verify: archive a pinned task from its page, then restore it and see it pinned again.
6. **Developer: docs and agent notes.**
   - Update `docs/using-the-app.md`, `docs/system-design.md`, `agents/implementation-notes.md` and `agents/memory-backend.md`.
   - Keep lines short and in the right audience folder.
   - Verify: `scripts/check-guardrails.sh` passes.

QA and Security checks follow the Developer steps, and DevOps then commits, pushes to `dev` and redeploys per AGENTS.md. Each step leaves the app working.

**Open questions:** None blocking. I am assuming running tasks are stopped automatically on archive or delete, with a warning in the dialog. I am also assuming there is no auto-delete and that shared uploads are never removed.