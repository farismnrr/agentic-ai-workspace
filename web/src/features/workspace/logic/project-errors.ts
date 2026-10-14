import { ApiError } from '../../../shared/api/http';
import { projectsCopy } from '../content/projects-copy';

/** Turns a failed project request into a sentence a person can act on. */
export function describeProjectError(error: unknown): string {
  return (error instanceof ApiError ? projectsCopy.errors[error.code] : undefined) ?? projectsCopy.fallbackError;
}
