-- A task made in plan mode stops once the PRD is written and waits for the person's yes before anyone builds.
-- Tasks that already exist keep running as they did, so the column starts at 0 for them.
ALTER TABLE tasks ADD COLUMN plan_first INTEGER NOT NULL DEFAULT 0;
