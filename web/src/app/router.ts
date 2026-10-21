import { writable } from 'svelte/store';

export type SettingsTab = 'models' | 'persona' | 'team' | 'spaces' | 'connected' | 'integrations' | 'activity' | 'usage' | 'notifications' | 'account' | 'env';

export type Route =
  | { name: 'landing' }
  | { name: 'not-found' }
  | { name: 'forbidden' }
  | { name: 'login' }
  | { name: 'register' }
  | { name: 'consent' }
  | { name: 'home' }
  | { name: 'inbox'; taskId?: string }
  | { name: 'projects' }
  | { name: 'new-project' }
  | { name: 'sites' }
  | { name: 'extensions'; projectId?: string }
  | { name: 'project'; projectId: string }
  | { name: 'task'; projectId: string; taskId: string }
  | { name: 'settings'; tab: SettingsTab };

const settingsTabs: readonly string[] = ['models', 'persona', 'team', 'spaces', 'connected', 'integrations', 'activity', 'usage', 'notifications', 'account', 'env'];

const isSettingsTab = (value: string | undefined): value is SettingsTab => value !== undefined && settingsTabs.includes(value);

const staticRoutes: Record<string, Route> = {
  '/': { name: 'landing' },
  '/not-found': { name: 'not-found' },
  '/forbidden': { name: 'forbidden' },
  '/login': { name: 'login' },
  '/register': { name: 'register' },
  '/oauth/consent': { name: 'consent' },
  '/app': { name: 'home' },
  '/app/inbox': { name: 'inbox' },
  '/app/projects': { name: 'projects' },
  '/app/projects/new': { name: 'new-project' },
  '/app/sites': { name: 'sites' },
  '/app/extensions': { name: 'extensions' },
  '/app/settings': { name: 'settings', tab: 'models' },
};

const appPatterns: [RegExp, (match: string[]) => Route | undefined][] = [
  [/^\/app\/inbox\/([^/]+)$/, ([, taskId]) => (taskId === undefined ? undefined : { name: 'inbox', taskId })],
  [/^\/app\/extensions\/([^/]+)$/, ([, projectId]) => (projectId === undefined ? undefined : { name: 'extensions', projectId })],
  [/^\/app\/projects\/([^/]+)$/, ([, projectId]) => (projectId === undefined ? undefined : { name: 'project', projectId })],
  [
    /^\/app\/projects\/([^/]+)\/tasks\/([^/]+)$/,
    ([, projectId, taskId]) => (projectId === undefined || taskId === undefined ? undefined : { name: 'task', projectId, taskId }),
  ],
  [/^\/app\/settings\/([^/]+)$/, ([, tab]) => (isSettingsTab(tab) ? { name: 'settings', tab } : undefined)],
];

function matchApp(path: string): Route | undefined {
  for (const [pattern, build] of appPatterns) {
    const match = pattern.exec(path);
    const route = match === null ? undefined : build(match);
    if (route !== undefined) return route;
  }
  return undefined;
}

/** Returns the route for a path, or undefined when the path is not an app route. */
export function matchRoute(path: string): Route | undefined {
  const clean = path.replace(/(.)\/$/, '$1');
  const fixed = staticRoutes[clean];
  if (fixed !== undefined) return fixed;
  return matchApp(clean);
}

const pathOf = (url: string): string => url.split(/[?#]/)[0] ?? url;

const resolve = (url: string): Route => matchRoute(pathOf(url)) ?? { name: 'not-found' };

export const route = writable<Route>(resolve(window.location.pathname));

/**
 * Goes to a same-site path after sign-in. App routes stay inside the page; anything else, such as an
 * app under /run, is a real page load because the server (not the SPA) answers it. `replace` leaves no
 * history entry behind, so Back does not return to the page that was left.
 */
export function goTo(path: string, replace = false): void {
  if (matchRoute(pathOf(path)) !== undefined) navigate(path, replace);
  else if (replace) window.location.replace(path);
  else window.location.assign(path);
}

/** Moves to a same-site URL (query allowed); `replace` skips adding a history entry. */
export function navigate(path: string, replace = false): void {
  if (replace) window.history.replaceState({}, '', path);
  else window.history.pushState({}, '', path);
  route.set(resolve(path));
  window.scrollTo(0, 0);
}

export function syncFromLocation(): void {
  route.set(resolve(window.location.pathname));
}

const isPlainClick = (event: MouseEvent): boolean =>
  !event.defaultPrevented && event.button === 0 && !(event.metaKey || event.ctrlKey || event.shiftKey || event.altKey);

const internalAnchor = (event: MouseEvent): HTMLAnchorElement | undefined => {
  const anchor = event.target instanceof Element ? event.target.closest('a') : null;
  return anchor?.target === '' && anchor.origin === window.location.origin ? anchor : undefined;
};

/** Turns plain same-origin clicks on known routes into client-side navigation. */
export function interceptLinkClick(event: MouseEvent): void {
  const anchor = isPlainClick(event) ? internalAnchor(event) : undefined;
  if (anchor?.hash === '' && matchRoute(anchor.pathname) !== undefined) {
    event.preventDefault();
    navigate(anchor.pathname);
  }
}
