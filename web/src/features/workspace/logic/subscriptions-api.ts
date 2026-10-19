import { ApiError, request } from '../../../shared/api/http';

/** The vendor programs a person can sign in to with their own plan. */
export type Program = 'claude' | 'chatgpt';

/** What the person must do in their browser: open the address and, for some programs, type the code. */
export interface SignInPrompt {
  url: string;
  code?: string;
}

/** A choice the program takes for its model, with the effort levels that go with it. */
export interface PlanModel {
  id: string;
  name: string;
  efforts: string[];
}

export interface SubscriptionState {
  connected: boolean;
  /** What the person can choose for this program's model. */
  models: PlanModel[];
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

const programs: Program[] = ['claude', 'chatgpt'];

/** A program the person is signed in to, with the choices it takes. */
export interface ConnectedPlan {
  program: Program;
  models: PlanModel[];
}

/** The programs the person is signed in to. */
export async function loadConnectedPlans(): Promise<ConnectedPlan[]> {
  const states = await Promise.all(programs.map(loadSubscription));
  return programs.flatMap((program, index) => {
    const state = states[index];
    return state?.connected === true ? [{ program, models: state.models }] : [];
  });
}
