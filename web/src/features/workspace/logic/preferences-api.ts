import { request } from '../../../shared/api/http';

export interface Rule {
  id: string;
  text: string;
  on: boolean;
}

export interface Preferences {
  defaultProviderId: string;
  defaultModel: string;
  /** How hard the AI thinks when a task does not say; empty is the provider's default. */
  defaultEffort: string;
  personaName: string;
  talkStyle: string;
  language: string;
  about: string;
  rules: Rule[];
}

export const loadPreferences = (): Promise<Preferences> => request<Preferences>('GET', '/api/preferences');

export const savePreferences = (preferences: Preferences): Promise<Preferences> => request<Preferences>('PUT', '/api/preferences', preferences);
