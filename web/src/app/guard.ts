import type { Route } from './router';

const openRoutes: readonly Route['name'][] = ['landing', 'login', 'register', 'not-found', 'forbidden'];

/** Everything except the public pages needs a signed-in person. */
export const requiresSession = (route: Route): boolean => !openRoutes.includes(route.name);

export const loginUrl = (returnTo: string): string => `/login?next=${encodeURIComponent(returnTo)}`;
