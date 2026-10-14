import { ApiError } from '../../../shared/api/http';
import { sitesCopy } from '../content/sites-copy';

/** Turns a failed request about an app into a sentence a person can act on. */
export function describeSiteError(error: unknown): string {
  return (error instanceof ApiError ? sitesCopy.errors[error.code] : undefined) ?? sitesCopy.fallbackError;
}
