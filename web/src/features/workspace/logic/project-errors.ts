import { ApiError } from '../../../shared/api/http';
import { projectsCopy } from '../content/projects-copy';

/** Turns a failed project request into a sentence a person can act on. */
export function describeProjectError(error: unknown): string {
  if (error instanceof ApiError) {
    return projectsCopy.errors[error.code] ?? error.detail ?? error.title ?? error.code;
  }
  return error instanceof Error ? error.message : projectsCopy.fallbackError;
}
