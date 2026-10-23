import { request } from '../../../shared/api/http';

export type ToolGroup = 'read' | 'write' | 'exec';

export interface SpaceTool {
  name: string;
  title: string;
  description: string;
  scope: string;
  group: ToolGroup;
  readOnly: boolean;
  enabled: boolean;
}

export interface ConnectorSpace {
  slug: string;
  name: string;
  path: string;
  url: string;
  scopes: ConnectedScope[];
  tools: SpaceTool[];
  skipApprovals: boolean;
  supportsApprovals: boolean;
}

export interface ConnectedScope {
  id: string;
  description: string;
}

export interface ConnectedApp {
  id: string;
  clientName: string;
  redirectHost: string;
  resourceName: string;
  scopes: ConnectedScope[];
  createdAt: string;
  lastUsedAt: string | null;
}

export const loadSpaces = (): Promise<ConnectorSpace[]> => request<ConnectorSpace[]>('GET', '/api/spaces');

export const loadConnectedApps = (): Promise<ConnectedApp[]> => request<ConnectedApp[]>('GET', '/api/oauth/grants');

export const revokeApp = (id: string): Promise<undefined> => request<undefined>('DELETE', `/api/oauth/grants/${encodeURIComponent(id)}`);

/** Lets the AI use every enabled tool of a space without asking; a tool that is off stays off. */
export const setSkipApprovals = (slug: string, skip: boolean): Promise<undefined> =>
  request<undefined>('PUT', `/api/spaces/${encodeURIComponent(slug)}/approvals`, { skip });

export const setToolEnabled = (slug: string, tool: string, enabled: boolean): Promise<undefined> =>
  request<undefined>('PUT', `/api/spaces/${encodeURIComponent(slug)}/tools/${encodeURIComponent(tool)}`, { enabled });

export type ActivityOutcome = 'ok' | 'failed' | 'off' | 'forbidden';

export interface ActivityEntry {
  id: string;
  at: string;
  space: string;
  /** The app that made the call, such as ChatGPT. */
  client: string;
  tool: string;
  summary: string;
  outcome: ActivityOutcome;
  detail: string;
  canUndo: boolean;
  undone: boolean;
}

export const loadActivity = (): Promise<ActivityEntry[]> => request<ActivityEntry[]>('GET', '/api/activity');

export const undoActivity = (id: string): Promise<undefined> =>
  request<undefined>('POST', `/api/activity/${encodeURIComponent(id)}/undo`);
