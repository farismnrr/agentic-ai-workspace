import { ApiError } from '../../../shared/api/http';
import { modelsCopy } from '../content/models';

/** Turns a failed provider request into a sentence a person can act on. */
export function describeProviderError(error: unknown): string {
  if (error instanceof ApiError) {
    return modelsCopy.errors[error.code] ?? error.detail ?? error.title ?? error.code;
  }
  return error instanceof Error ? error.message : modelsCopy.fallbackError;
}
