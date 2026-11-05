ALTER TABLE task_delegations ADD COLUMN execution_context TEXT NOT NULL DEFAULT '';
ALTER TABLE task_delegations ADD COLUMN request_effort TEXT NOT NULL DEFAULT '';
ALTER TABLE task_delegations ADD COLUMN execution_scope TEXT NOT NULL DEFAULT '[]';
ALTER TABLE task_delegations ADD COLUMN mutating INTEGER NOT NULL DEFAULT 0;
