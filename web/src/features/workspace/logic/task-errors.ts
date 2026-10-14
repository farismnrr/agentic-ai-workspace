import { ApiError } from '../../../shared/api/http';
import { taskErrors, taskFallbackError } from '../content/tasks-copy';

/** Turns a failed task request into a sentence a person can act on. */
export function describeTaskError(error: unknown): string {
  return (error instanceof ApiError ? taskErrors[error.code] : undefined) ?? taskFallbackError;
}
