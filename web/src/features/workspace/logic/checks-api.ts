import { request } from '../../../shared/api/http';

export interface Check {
  name: string;
  description: string;
  command: string[];
  timeoutSeconds: number;
}

const path = (projectId: string): string => `/api/projects/${encodeURIComponent(projectId)}/checks`;

export const loadChecks = (projectId: string): Promise<Check[]> => request<Check[]>('GET', path(projectId));

export const saveChecks = (projectId: string, checks: Check[]): Promise<undefined> => request<undefined>('PUT', path(projectId), checks);
