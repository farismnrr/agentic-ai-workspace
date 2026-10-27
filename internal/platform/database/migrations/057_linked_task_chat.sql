ALTER TABLE tasks ADD COLUMN linked_task_id TEXT REFERENCES tasks(id) ON DELETE SET NULL;
ALTER TABLE tasks ADD COLUMN origin_task_id TEXT REFERENCES tasks(id) ON DELETE SET NULL;

CREATE UNIQUE INDEX tasks_linked_task_unique ON tasks(linked_task_id) WHERE linked_task_id IS NOT NULL;
CREATE UNIQUE INDEX tasks_origin_task_unique ON tasks(origin_task_id)
    WHERE origin_task_id IS NOT NULL AND archived_at IS NULL;

CREATE TRIGGER tasks_linked_chat_same_project
BEFORE UPDATE OF linked_task_id ON tasks
WHEN NEW.linked_task_id IS NOT NULL AND NOT EXISTS (
    SELECT 1 FROM tasks linked
    WHERE linked.id = NEW.linked_task_id
      AND linked.project_id = NEW.project_id
      AND linked.user_id = NEW.user_id
      AND linked.origin_task_id = NEW.id
      AND linked.source = ''
      AND linked.work_id = ''
)
BEGIN
    SELECT RAISE(ABORT, 'linked chat must be an internal task in the same project');
END;
