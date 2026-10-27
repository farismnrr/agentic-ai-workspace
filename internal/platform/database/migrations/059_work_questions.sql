CREATE TABLE external_work_questions (
    id           TEXT PRIMARY KEY,
    work_id      TEXT NOT NULL REFERENCES external_works(id) ON DELETE CASCADE,
    task_id      TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id   TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    seq          INTEGER NOT NULL,
    request_key  TEXT NOT NULL DEFAULT '',
    form         TEXT NOT NULL,
    state        TEXT NOT NULL CHECK (state IN ('open', 'answered', 'cancelled')),
    answers      TEXT NOT NULL DEFAULT '[]',
    summary      TEXT NOT NULL DEFAULT '',
    event_id     TEXT NOT NULL REFERENCES task_events(id) ON DELETE CASCADE,
    asked_at     INTEGER NOT NULL,
    answered_at  INTEGER,
    answered_by  TEXT NOT NULL DEFAULT '',
    answer_source TEXT NOT NULL DEFAULT '',
    UNIQUE(work_id, seq)
);

CREATE UNIQUE INDEX external_work_question_request_idx
    ON external_work_questions(work_id, request_key) WHERE request_key <> '';
CREATE INDEX external_work_question_state_idx
    ON external_work_questions(work_id, state, seq);
