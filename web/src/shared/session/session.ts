import { writable } from 'svelte/store';
import { request } from '../api/http';

export interface SessionUser {
  id: string;
  name: string;
  email: string;
}

export type SessionState = { status: 'loading' } | { status: 'anonymous' } | { status: 'signed-in'; user: SessionUser };

export const session = writable<SessionState>({ status: 'loading' });

/** Asks the backend who is signed in; any failure means anonymous. */
export async function loadSession(): Promise<void> {
  try {
    session.set({ status: 'signed-in', user: await request<SessionUser>('GET', '/api/auth/me') });
  } catch {
    session.set({ status: 'anonymous' });
  }
}

export function setUser(user: SessionUser): void {
  session.set({ status: 'signed-in', user });
}

export async function signOut(): Promise<void> {
  try {
    await request<undefined>('POST', '/api/auth/logout');
  } finally {
    session.set({ status: 'anonymous' });
  }
}
