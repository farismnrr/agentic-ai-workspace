-- The conversation a vendor's own coding program keeps for a task that runs on the person's own plan, so a
-- follow-up message continues it. The program holds the conversation; this is only its name.
CREATE TABLE task_cli_sessions (
    task_id TEXT PRIMARY KEY REFERENCES tasks (id) ON DELETE CASCADE,
    session TEXT NOT NULL
);
