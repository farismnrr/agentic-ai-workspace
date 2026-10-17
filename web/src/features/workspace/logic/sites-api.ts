import { request } from '../../../shared/api/http';

export type Visibility = 'private' | 'public';

export interface Site {
  projectId: string;
  /** Path the app is opened at, such as /run/my-shop/<uuid>/. */
  url: string;
  visibility: Visibility;
  status: 'running' | 'stopped';
  port: number;
  /** ISO timestamp. */
  startedAt: string;
}

const path = (projectId: string): string => `/api/sites/${encodeURIComponent(projectId)}`;

export const loadSites = async (): Promise<Site[]> => (await request<{ sites: Site[] }>('GET', '/api/sites')).sites;

export const stopSite = (projectId: string): Promise<undefined> => request<undefined>('POST', `${path(projectId)}/stop`);

export const startSite = (projectId: string): Promise<Site> => request<Site>('POST', `${path(projectId)}/start`);

export const setVisibility = (projectId: string, visibility: Visibility): Promise<undefined> =>
  request<undefined>('PATCH', path(projectId), { visibility });
