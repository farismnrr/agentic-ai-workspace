import { request } from '../../../shared/api/http';

export type AccessGroup = 'read' | 'write' | 'exec';

export interface TeamRole {
  /** Names the role to the lead; empty for a role not saved yet, which the server names. */
  key: string;
  name: string;
  purpose: string;
  /** The role's personality and way of working. */
  prompt: string;
  /** Which model the role uses; both empty means the default model. */
  providerId: string;
  model: string;
  effort: string;
  access: string[];
  lead: boolean;
  enabled: boolean;
  /** One of the built-in SDLC roles. */
  preset: boolean;
}

export interface AgentTeam {
  useByDefault: boolean;
  roles: TeamRole[];
}

export const loadAgentTeam = (): Promise<AgentTeam> => request<AgentTeam>('GET', '/api/agent-team');

export const saveAgentTeam = (team: AgentTeam): Promise<AgentTeam> => request<AgentTeam>('PUT', '/api/agent-team', team);

export const resetAgentTeam = (): Promise<AgentTeam> => request<AgentTeam>('POST', '/api/agent-team/reset');

export const newRole = (): TeamRole => ({
  key: '',
  name: '',
  purpose: '',
  prompt: '',
  providerId: '',
  model: '',
  effort: '',
  access: ['read'],
  lead: false,
  enabled: true,
  preset: false,
});

/** The role's model as the model picker writes it ("providerId::model"), or empty for the default model. */
export const modelPick = (role: TeamRole): string => (role.providerId === '' || role.model === '' ? '' : `${role.providerId}::${role.model}`);

export function applyPick(role: TeamRole, value: string): void {
  const [providerId = '', model = ''] = value.split('::');
  role.providerId = value === '' ? '' : providerId;
  role.model = value === '' ? '' : model;
}
