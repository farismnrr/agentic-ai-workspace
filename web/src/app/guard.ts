import type { Route } from './router';

const openRoutes: readonly Route['name'][] = ['landing', 'login', 'register', 'not-found', 'forbidden'];

const guestRoutes: readonly Route['name'][] = ['login', 'register'];

/** Pages for someone who is not signed in: a signed-in person is taken on to where they were going. */
export const guestOnly = (route: Route): boolean => guestRoutes.includes(route.name);

/** Everything except the public pages needs a signed-in person. */
export const requiresSession = (route: Route): boolean => !openRoutes.includes(route.name);

export const loginUrl = (returnTo: string): string => `/login?next=${encodeURIComponent(returnTo)}`;
