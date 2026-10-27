ALTER TABLE external_works ADD COLUMN plan_state TEXT NOT NULL DEFAULT 'draft'
    CHECK (plan_state IN ('draft', 'awaiting_clarification', 'ready_for_approval', 'approved', 'handed_off'));

CREATE TABLE external_plan_revisions (
    id TEXT PRIMARY KEY,
    work_id TEXT NOT NULL REFERENCES external_works(id) ON DELETE CASCADE,
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    rev_no INTEGER NOT NULL,
    plan_rev INTEGER NOT NULL,
    steps TEXT NOT NULL,
    prd TEXT NOT NULL,
    research TEXT NOT NULL,
    qa TEXT NOT NULL,
    small_task INTEGER NOT NULL DEFAULT 0 CHECK (small_task IN (0, 1)),
    summary TEXT NOT NULL,
    digest TEXT NOT NULL,
    event_id TEXT NOT NULL REFERENCES task_events(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('submitted', 'superseded', 'changes_requested', 'approved')),
    created_at INTEGER NOT NULL,
    UNIQUE(work_id, rev_no)
);

CREATE UNIQUE INDEX external_plan_revision_submitted
    ON external_plan_revisions(work_id) WHERE status = 'submitted';

CREATE TRIGGER external_plan_revision_content_immutable
BEFORE UPDATE OF id, work_id, task_id, rev_no, plan_rev, steps, prd, research, qa,
    small_task, summary, digest, event_id, created_at ON external_plan_revisions
BEGIN
    SELECT RAISE(ABORT, 'plan revision content is immutable');
END;
