-- A process restart cannot resume an in-flight summary operation. Reset any
-- old transient indicator; a future model/CLI usage event will measure it anew.
-- Do not claim the pre-compaction token count is the new active context.
UPDATE tasks
SET context_used = 0, context_window = 0, context_state = 'unknown'
WHERE context_state = 'compacting';
