import { request } from '../../../shared/api/http';
import type { Project } from './task-types';

export const loadProjects = (): Promise<Project[]> => request<Project[]>('GET', '/api/projects');

export const createProject = (name: string, description: string): Promise<Project> =>
  request<Project>('POST', '/api/projects', { name, description });

/** Clones a repository into an empty project. The server only accepts one the person let the app reach. */
export const importRepo = (id: string, repo: string): Promise<undefined> =>
  request<undefined>('POST', `/api/projects/${encodeURIComponent(id)}/import`, { repo });

/** Copies a template's look (never its backend) into the project. The server only accepts names it knows. */
export const applyTemplate = (id: string, template: string): Promise<undefined> =>
  request<undefined>('POST', `/api/projects/${encodeURIComponent(id)}/template`, { template });

export const updateProject = (id: string, name: string, description: string): Promise<Project> =>
  request<Project>('PUT', `/api/projects/${encodeURIComponent(id)}`, { name, description });

/** Deletes a project for good, with its files and tasks. There is no way back, so the page asks first. */
export const deleteProject = (id: string): Promise<undefined> =>
  request<undefined>('DELETE', `/api/projects/${encodeURIComponent(id)}/forever`);

export const archiveProject = (id: string): Promise<undefined> => request<undefined>('DELETE', `/api/projects/${encodeURIComponent(id)}`);
