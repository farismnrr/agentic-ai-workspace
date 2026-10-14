import { ApiError, request } from '../../../shared/api/http';
import type { SessionUser } from '../../../shared/session/session';
import { errorCopy, FALLBACK_ERROR } from '../content';

export const signIn = (email: string, password: string): Promise<SessionUser> =>
  request<SessionUser>('POST', '/api/auth/login', { email, password });

export const createAccount = (name: string, email: string, password: string): Promise<SessionUser> =>
  request<SessionUser>('POST', '/api/auth/register', { name, email, password });

/** Turns a failed auth request into a sentence a person can act on. */
export function describeError(error: unknown): string {
  return error instanceof ApiError ? (errorCopy[error.code] ?? FALLBACK_ERROR) : FALLBACK_ERROR;
}

/** Reads the `next` query parameter, accepting only same-site paths. */
export function nextPath(search: string): string {
  const next = new URLSearchParams(search).get('next');
  return next !== null && next.startsWith('/') && !next.startsWith('//') ? next : '/app';
}
