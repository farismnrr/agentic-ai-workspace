import type { TaskStatus, WorkSummary } from './task-types';

/** Shows activity only while the task can still make progress. */
export function shouldShowWorkingIndicator(status: TaskStatus, work?: WorkSummary): boolean {
  if (!['queued', 'planning', 'working'].includes(status)) return false;
  return work === undefined || (work.state === 'open' && !work.stale);
}
