-- How full the model's context window was at the task's last model call, and how big the window is,
-- so the task page can show it. Zero means not known yet.
ALTER TABLE tasks ADD COLUMN context_used   INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tasks ADD COLUMN context_window INTEGER NOT NULL DEFAULT 0;
