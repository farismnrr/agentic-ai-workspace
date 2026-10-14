import { ApiError } from '../../../shared/api/http';
import { modelsCopy } from '../content/models';

/** Turns a failed provider request into a sentence a person can act on. */
export function describeProviderError(error: unknown): string {
  return (error instanceof ApiError ? modelsCopy.errors[error.code] : undefined) ?? modelsCopy.fallbackError;
}
