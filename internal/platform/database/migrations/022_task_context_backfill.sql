-- Tasks that ran before the context meter existed: take the size of their last model call.
UPDATE tasks
   SET context_used = COALESCE(
         (SELECT input_tokens + output_tokens FROM task_usage WHERE task_usage.task_id = tasks.id
           ORDER BY task_usage.id DESC LIMIT 1), 0)
 WHERE context_used = 0;
