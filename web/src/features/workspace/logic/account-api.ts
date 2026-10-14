import { ApiError, request } from '../../../shared/api/http';

export const changePassword = (current: string, next: string): Promise<undefined> =>
  request<undefined>('POST', '/api/auth/password', { current, next });

const passwordErrors: Record<string, string> = {
  wrong_password: 'That password is not right.',
  weak_password: 'Use between 8 and 128 characters.',
  too_many_requests: 'Too many attempts. Please wait a few minutes and try again.',
};

/** Turns a failed password change into a sentence a person can act on. */
export const describeAccountError = (error: unknown): string =>
  error instanceof ApiError ? (passwordErrors[error.code] ?? 'Something went wrong. Please try again.') : 'Something went wrong. Please try again.';

export const deleteAllTasks = (): Promise<undefined> => request<undefined>('DELETE', '/api/tasks');

export const deleteAccount = (password: string): Promise<undefined> =>
  request<undefined>('POST', '/api/auth/delete', { password });
