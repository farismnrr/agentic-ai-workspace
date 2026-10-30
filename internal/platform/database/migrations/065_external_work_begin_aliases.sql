-- A repeated begin_work with a new request key can resume the same issue.
-- Keep that key idempotent even after the issue is completed or reconnected.
CREATE TABLE external_work_begin_aliases (
    user_id TEXT NOT NULL, client TEXT NOT NULL, project_id TEXT NOT NULL,
    idem_key TEXT NOT NULL, idem_hash TEXT NOT NULL,
    work_id TEXT NOT NULL REFERENCES external_works(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, client, project_id, idem_key)
);
