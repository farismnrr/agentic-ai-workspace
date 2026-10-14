import { request } from '../../../shared/api/http';
import type { Project } from './task-types';

export const loadProjects = (): Promise<Project[]> => request<Project[]>('GET', '/api/projects');

export const createProject = (name: string, description: string): Promise<Project> =>
  request<Project>('POST', '/api/projects', { name, description });

export const updateProject = (id: string, name: string, description: string): Promise<Project> =>
  request<Project>('PUT', `/api/projects/${encodeURIComponent(id)}`, { name, description });

export const archiveProject = (id: string): Promise<undefined> => request<undefined>('DELETE', `/api/projects/${encodeURIComponent(id)}`);
