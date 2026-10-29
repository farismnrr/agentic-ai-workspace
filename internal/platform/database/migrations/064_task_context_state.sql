-- Live context state separate from the token count. Retain measured history.
ALTER TABLE tasks ADD COLUMN context_state TEXT NOT NULL DEFAULT 'unknown'
    CHECK(context_state IN ('unknown', 'measured', 'compacting', 'failed'));
UPDATE tasks SET context_state = 'measured' WHERE context_used > 0 AND context_window > 0;
