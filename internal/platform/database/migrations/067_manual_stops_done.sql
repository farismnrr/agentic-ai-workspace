-- Historical tasks explicitly stopped by their owner should not remain in
-- Needs Attention. Preserve the original work state, proof and event history.
UPDATE tasks SET status = 'done'
WHERE status = 'failed' AND EXISTS (
    SELECT 1 FROM external_works work
    WHERE work.id = tasks.work_id AND work.state = 'stopped'
);

-- Old queued manual cancellations have an unambiguous recorded stop reason.
-- Do not rewrite generic "Stopped." failures: they can also mean a process shutdown.
UPDATE tasks SET status = 'done'
WHERE status = 'failed' AND work_id = '' AND (
    SELECT json_extract(body, '$.text') FROM task_events
    WHERE task_id = tasks.id ORDER BY seq DESC LIMIT 1
) = 'Stopped before I started.';
