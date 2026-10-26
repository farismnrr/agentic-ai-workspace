package sqlite

const testWorkSchema = `
PRAGMA foreign_keys = ON;
CREATE TABLE users (id TEXT PRIMARY KEY);
CREATE TABLE projects (id TEXT PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users(id));
CREATE TABLE tasks (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), user_id TEXT NOT NULL REFERENCES users(id),
 title TEXT NOT NULL, summary TEXT NOT NULL DEFAULT '', status TEXT NOT NULL, mode TEXT NOT NULL,
 provider_id TEXT NOT NULL, model TEXT NOT NULL, plan TEXT NOT NULL DEFAULT '[]', proof TEXT,
 created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL, source TEXT NOT NULL DEFAULT '',
 context_used INTEGER NOT NULL DEFAULT 0, context_window INTEGER NOT NULL DEFAULT 0,
 effort TEXT NOT NULL DEFAULT '', team INTEGER NOT NULL DEFAULT 0, prd TEXT NOT NULL DEFAULT '',
 research TEXT NOT NULL DEFAULT '', plan_first INTEGER NOT NULL DEFAULT 0,
 follow_settings INTEGER NOT NULL DEFAULT 0, pinned_at INTEGER, archived_at INTEGER,
 work_id TEXT NOT NULL DEFAULT ''
);
CREATE TABLE task_events (
 task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE, seq INTEGER NOT NULL,
 id TEXT NOT NULL UNIQUE, at INTEGER NOT NULL, kind TEXT NOT NULL, tool_use_id TEXT NOT NULL DEFAULT '',
 body TEXT NOT NULL, undo BLOB, PRIMARY KEY(task_id, seq)
);
CREATE TABLE external_works (
 id TEXT PRIMARY KEY, task_id TEXT NOT NULL UNIQUE REFERENCES tasks(id) ON DELETE CASCADE,
 user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE, client TEXT NOT NULL,
 state TEXT NOT NULL, role_key TEXT NOT NULL DEFAULT '', role_name TEXT NOT NULL DEFAULT '',
 rev INTEGER NOT NULL DEFAULT 0, plan_rev INTEGER NOT NULL DEFAULT 0, next_step INTEGER NOT NULL DEFAULT 1,
 idem_key TEXT NOT NULL DEFAULT '', idem_hash TEXT NOT NULL DEFAULT '', created_at INTEGER NOT NULL,
 updated_at INTEGER NOT NULL, ended_at INTEGER
);
CREATE UNIQUE INDEX external_works_idempotency_idx
 ON external_works(user_id,client,project_id,idem_key) WHERE idem_key <> '';
CREATE TABLE external_work_ops (
 work_id TEXT NOT NULL REFERENCES external_works(id) ON DELETE CASCADE, op_key TEXT NOT NULL,
 tool TEXT NOT NULL, req_hash TEXT NOT NULL, response BLOB NOT NULL, at INTEGER NOT NULL,
 PRIMARY KEY(work_id,op_key)
);
`
