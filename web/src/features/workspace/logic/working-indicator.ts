import type { TaskStatus, WorkSummary } from './task-types';
import { workStatusPresentation } from './work-status';

/** Uses the shared lifecycle rules for the thread's activity indicator. */
export function shouldShowWorkingIndicator(status: TaskStatus, work?: WorkSummary): boolean {
  return workStatusPresentation(status, work).showWorking;
}
