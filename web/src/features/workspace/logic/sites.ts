import type { Site } from './sites-api';
import type { Project } from './task-types';

/** Apps that are up right now, newest first as the server sends them. */
export const runningSites = (sites: Site[]): Site[] => sites.filter((site) => site.status === 'running');

/** Running apps first, then stopped ones; the order inside each group is kept. */
export const orderedSites = (sites: Site[]): Site[] => [...runningSites(sites), ...sites.filter((site) => site.status !== 'running')];

/** The name of the project an app belongs to. */
export const siteName = (site: Site, projects: Project[]): string => projects.find((project) => project.id === site.projectId)?.name ?? 'Your app';

/** The full address of an app, for copying. */
export const absoluteUrl = (site: Site): string => `${window.location.origin}${site.url}`;
