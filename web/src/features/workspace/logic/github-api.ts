import { request } from '../../../shared/api/http';

export interface GitHubStatus {
  configured: boolean;
  connected: boolean;
  login: string;
  name: string;
  /** Where to choose which repositories the app may reach; empty when the server does not know. */
  reposUrl: string;
}

export interface GitHubRepo {
  /** owner/name */
  name: string;
  description: string;
  private: boolean;
}

/** The repositories the person let the GitHub App reach, newest first. */
export const loadRepos = async (): Promise<GitHubRepo[]> => (await request<{ repos: GitHubRepo[] }>('GET', '/api/integrations/github/repos')).repos;

export const loadGitHub = (): Promise<GitHubStatus> => request<GitHubStatus>('GET', '/api/integrations/github');

/** Returns the GitHub address where the person approves the connection. */
export const startGitHub = async (): Promise<string> => (await request<{ url: string }>('POST', '/api/integrations/github/start')).url;

export const disconnectGitHub = (): Promise<undefined> => request<undefined>('DELETE', '/api/integrations/github');
