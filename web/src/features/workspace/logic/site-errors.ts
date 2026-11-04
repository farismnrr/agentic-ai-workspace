import { ApiError } from '../../../shared/api/http';
import { sitesCopy } from '../content/sites-copy';

/** Turns a failed request about an app into a sentence a person can act on. */
export function describeSiteError(error: unknown): string {
  if (error instanceof ApiError) {
    return sitesCopy.errors[error.code] ?? error.detail ?? error.title ?? error.code;
  }
  return error instanceof Error ? error.message : sitesCopy.fallbackError;
}
