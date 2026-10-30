export type SettingsTab = 'models' | 'persona' | 'team' | 'spaces' | 'connected' | 'integrations' | 'activity' | 'usage' | 'notifications' | 'database' | 'account';

export type Route =
  | { name: 'not-found' }
  | { name: 'forbidden' }
  | { name: 'login' }
  | { name: 'register' }
  | { name: 'consent' }
  | { name: 'home' }
  | { name: 'tasks' }
  | { name: 'archive' }
  | { name: 'inbox'; taskId?: string }
  | { name: 'projects' }
  | { name: 'new-project' }
  | { name: 'sites' }
  | { name: 'extensions'; projectId?: string }
  | { name: 'project'; projectId: string }
  | { name: 'task'; projectId: string; taskId: string }
  | { name: 'settings'; tab: SettingsTab };

const settingsTabs: readonly string[] = ['models', 'persona', 'team', 'spaces', 'connected', 'integrations', 'activity', 'usage', 'notifications', 'database', 'account'];
const isSettingsTab = (value: string | undefined): value is SettingsTab => value !== undefined && settingsTabs.includes(value);

const staticRoutes: Record<string, Route> = {
  '/': { name: 'login' }, '/not-found': { name: 'not-found' }, '/forbidden': { name: 'forbidden' },
  '/login': { name: 'login' }, '/register': { name: 'register' }, '/oauth/consent': { name: 'consent' },
  '/app': { name: 'home' }, '/app/tasks': { name: 'tasks' }, '/app/archive': { name: 'archive' },
  '/app/inbox': { name: 'inbox' }, '/app/projects': { name: 'projects' }, '/app/projects/new': { name: 'new-project' },
  '/app/sites': { name: 'sites' }, '/app/extensions': { name: 'extensions' },
  '/app/settings': { name: 'settings', tab: 'models' },
};

const appPatterns: [RegExp, (match: string[]) => Route | undefined][] = [
  [/^\/app\/inbox\/([^/]+)$/, ([, taskId]) => taskId === undefined ? undefined : { name: 'inbox', taskId }],
  [/^\/app\/extensions\/([^/]+)$/, ([, projectId]) => projectId === undefined ? undefined : { name: 'extensions', projectId }],
  [/^\/app\/projects\/([^/]+)$/, ([, projectId]) => projectId === undefined ? undefined : { name: 'project', projectId }],
  [/^\/app\/projects\/([^/]+)\/tasks\/([^/]+)$/, ([, projectId, taskId]) =>
    projectId === undefined || taskId === undefined ? undefined : { name: 'task', projectId, taskId }],
  [/^\/app\/settings\/([^/]+)$/, ([, tab]) => isSettingsTab(tab) ? { name: 'settings', tab } : undefined],
];

/** Matches known application paths without browser state so navigation can be tested directly. */
export function matchRoute(path: string): Route | undefined {
  const clean = path.replace(/(.)\/$/, '$1');
  const fixed = staticRoutes[clean];
  if (fixed !== undefined) return fixed;
  for (const [pattern, build] of appPatterns) {
    const match = pattern.exec(clean);
    const route = match === null ? undefined : build(match);
    if (route !== undefined) return route;
  }
  return undefined;
}
