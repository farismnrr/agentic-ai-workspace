import { ApiError, request } from '../../../shared/api/http';

/** The vendor programs a person can sign in to with their own plan. */
export type Program = 'claude' | 'chatgpt';

/** What the person must do in their browser: open the address and, for some programs, type the code. */
export interface SignInPrompt {
  url: string;
  code?: string;
}

export interface SubscriptionState {
  connected: boolean;
  /** Set while a sign-in waits for the person to finish. */
  prompt?: SignInPrompt;
}

const path = (program: Program): string => `/api/subscriptions/${program}`;

export const loadSubscription = (program: Program): Promise<SubscriptionState> => request<SubscriptionState>('GET', path(program));

export const startSignIn = (program: Program): Promise<SignInPrompt> => request<SignInPrompt>('POST', `${path(program)}/start`);

/** Gives the program the code the vendor showed the person. */
export const submitSignInCode = (program: Program, code: string): Promise<undefined> => request<undefined>('POST', `${path(program)}/code`, { code });

export const signOut = (program: Program): Promise<undefined> => request<undefined>('DELETE', path(program));

/** Plain words for why a sign-in step failed. */
export function describeSignInError(failure: unknown, copy: { badCode: string; failed: string; unavailable: string }): string {
  if (!(failure instanceof ApiError)) return copy.failed;
  if (failure.code === 'bad_code') return copy.badCode;
  if (failure.code === 'unavailable') return copy.unavailable;
  return copy.failed;
}
