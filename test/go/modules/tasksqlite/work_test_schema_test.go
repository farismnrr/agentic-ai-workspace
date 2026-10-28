package tasksqlite_test

const testWorkSchema = `
PRAGMA foreign_keys = ON;
CREATE TABLE users (id TEXT PRIMARY KEY);
CREATE TABLE projects (id TEXT PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users(id));
CREATE TABLE tasks (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), user_id TEXT NOT NULL REFERENCES users(id),
 title TEXT NOT NULL, summary TEXT NOT NULL DEFAULT '', status TEXT NOT NULL, mode TEXT NOT NULL,
 review_mode TEXT NOT NULL DEFAULT 'swift' CHECK (review_mode IN ('swift', 'deep-review')),
 provider_id TEXT NOT NULL, model TEXT NOT NULL, plan TEXT NOT NULL DEFAULT '[]', proof TEXT,
 created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL, source TEXT NOT NULL DEFAULT '',
 context_used INTEGER NOT NULL DEFAULT 0, context_window INTEGER NOT NULL DEFAULT 0,
 effort TEXT NOT NULL DEFAULT '', team INTEGER NOT NULL DEFAULT 0, prd TEXT NOT NULL DEFAULT '',
 research TEXT NOT NULL DEFAULT '', plan_first INTEGER NOT NULL DEFAULT 0,
 follow_settings INTEGER NOT NULL DEFAULT 0, pinned_at INTEGER, archived_at INTEGER,
 work_id TEXT NOT NULL DEFAULT '', linked_task_id TEXT, origin_task_id TEXT
);
CREATE UNIQUE INDEX tasks_linked_task_unique ON tasks(linked_task_id) WHERE linked_task_id IS NOT NULL;
CREATE UNIQUE INDEX tasks_origin_task_unique ON tasks(origin_task_id)
 WHERE origin_task_id IS NOT NULL AND archived_at IS NULL;
CREATE TRIGGER tasks_linked_chat_same_project
BEFORE UPDATE OF linked_task_id ON tasks
WHEN NEW.linked_task_id IS NOT NULL AND NOT EXISTS (
 SELECT 1 FROM tasks linked WHERE linked.id=NEW.linked_task_id AND linked.project_id=NEW.project_id
 AND linked.user_id=NEW.user_id AND linked.origin_task_id=NEW.id AND linked.source='' AND linked.work_id=''
)
BEGIN SELECT RAISE(ABORT, 'linked chat must be an internal task in the same project'); END;
CREATE TABLE task_events (
 task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE, seq INTEGER NOT NULL,
 id TEXT NOT NULL UNIQUE, at INTEGER NOT NULL, kind TEXT NOT NULL, tool_use_id TEXT NOT NULL DEFAULT '',
	 body TEXT NOT NULL, undo BLOB, version INTEGER NOT NULL DEFAULT 1, PRIMARY KEY(task_id, seq)
);
CREATE TABLE task_transcript (
 task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE, seq INTEGER NOT NULL,
 role TEXT NOT NULL, block TEXT NOT NULL, scope TEXT NOT NULL DEFAULT '', PRIMARY KEY(task_id, seq, scope)
);
CREATE TABLE external_works (
 id TEXT PRIMARY KEY, task_id TEXT NOT NULL UNIQUE REFERENCES tasks(id) ON DELETE CASCADE,
 user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE, client TEXT NOT NULL,
 state TEXT NOT NULL, role_key TEXT NOT NULL DEFAULT '', role_name TEXT NOT NULL DEFAULT '',
 rev INTEGER NOT NULL DEFAULT 0, plan_rev INTEGER NOT NULL DEFAULT 0, next_step INTEGER NOT NULL DEFAULT 1,
 idem_key TEXT NOT NULL DEFAULT '', idem_hash TEXT NOT NULL DEFAULT '', created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL, ended_at INTEGER, reported_at INTEGER,
 plan_state TEXT NOT NULL DEFAULT 'draft'
);
CREATE TABLE external_plan_revisions (
 id TEXT PRIMARY KEY, work_id TEXT NOT NULL REFERENCES external_works(id) ON DELETE CASCADE,
 task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE, rev_no INTEGER NOT NULL,
 plan_rev INTEGER NOT NULL, steps TEXT NOT NULL, prd TEXT NOT NULL, research TEXT NOT NULL,
 qa TEXT NOT NULL, small_task INTEGER NOT NULL DEFAULT 0, summary TEXT NOT NULL, digest TEXT NOT NULL,
 event_id TEXT NOT NULL REFERENCES task_events(id) ON DELETE CASCADE,
 status TEXT NOT NULL, created_at INTEGER NOT NULL, UNIQUE(work_id, rev_no)
);
CREATE UNIQUE INDEX external_plan_revision_submitted ON external_plan_revisions(work_id) WHERE status='submitted';
CREATE TRIGGER external_plan_revision_content_immutable
BEFORE UPDATE OF id, work_id, task_id, rev_no, plan_rev, steps, prd, research, qa,
 small_task, summary, digest, event_id, created_at ON external_plan_revisions
BEGIN SELECT RAISE(ABORT, 'plan revision content is immutable'); END;
CREATE TABLE external_work_questions (
 id TEXT PRIMARY KEY, work_id TEXT NOT NULL REFERENCES external_works(id) ON DELETE CASCADE,
 task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
 user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
 seq INTEGER NOT NULL, request_key TEXT NOT NULL DEFAULT '', form TEXT NOT NULL,
 state TEXT NOT NULL CHECK(state IN ('open','answered','cancelled')),
 answers TEXT NOT NULL DEFAULT '[]', summary TEXT NOT NULL DEFAULT '',
 event_id TEXT NOT NULL REFERENCES task_events(id) ON DELETE CASCADE, asked_at INTEGER NOT NULL,
 answered_at INTEGER, answered_by TEXT NOT NULL DEFAULT '', answer_source TEXT NOT NULL DEFAULT '',
 UNIQUE(work_id, seq)
);
CREATE UNIQUE INDEX external_work_question_request_idx ON external_work_questions(work_id,request_key)
 WHERE request_key <> '';
CREATE TABLE external_plan_decisions (
 id TEXT PRIMARY KEY, revision_id TEXT NOT NULL UNIQUE REFERENCES external_plan_revisions(id) ON DELETE CASCADE,
 work_id TEXT NOT NULL REFERENCES external_works(id) ON DELETE CASCADE,
 task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
 user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
 decision TEXT NOT NULL, source TEXT NOT NULL, client TEXT NOT NULL DEFAULT '', idem_key TEXT NOT NULL DEFAULT '',
 req_hash TEXT NOT NULL, note TEXT NOT NULL DEFAULT '', decided_at INTEGER NOT NULL
);
CREATE UNIQUE INDEX external_plan_decision_idempotency_idx
 ON external_plan_decisions(work_id,idem_key) WHERE idem_key <> '';
CREATE TABLE external_plan_handoffs (
 id TEXT PRIMARY KEY, decision_id TEXT NOT NULL UNIQUE REFERENCES external_plan_decisions(id) ON DELETE CASCADE,
 work_id TEXT NOT NULL UNIQUE REFERENCES external_works(id) ON DELETE CASCADE,
 state TEXT NOT NULL, attempts INTEGER NOT NULL DEFAULT 0, next_attempt_at INTEGER NOT NULL,
 last_error TEXT NOT NULL DEFAULT '', internal_task_id TEXT REFERENCES tasks(id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX external_plan_handoff_task_idx ON external_plan_handoffs(internal_task_id)
 WHERE internal_task_id IS NOT NULL;
CREATE INDEX external_plan_handoff_due_idx ON external_plan_handoffs(state,next_attempt_at);
CREATE UNIQUE INDEX external_works_idempotency_idx
 ON external_works(user_id,client,project_id,idem_key) WHERE idem_key <> '';
CREATE TABLE external_work_ops (
 work_id TEXT NOT NULL REFERENCES external_works(id) ON DELETE CASCADE, op_key TEXT NOT NULL,
 tool TEXT NOT NULL, req_hash TEXT NOT NULL, response BLOB NOT NULL, at INTEGER NOT NULL,
 PRIMARY KEY(work_id,op_key)
);
`
