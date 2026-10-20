import { request } from '../../../shared/api/http';

export interface Rule {
  id: string;
  text: string;
  on: boolean;
}

/** A model and how hard it thinks, set once for a tier that roles choose from. */
export interface ModelTier {
  providerId: string;
  model: string;
  effort: string;
}

export interface Preferences {
  defaultProviderId: string;
  defaultModel: string;
  /** How hard the AI thinks when a task does not say; empty is the provider's default. */
  defaultEffort: string;
  /** The model for roles on the Thinking tier; empty means the default model. */
  thinking: ModelTier;
  /** The model for roles on the Fast tier; empty means the default model. */
  fast: ModelTier;
  personaName: string;
  talkStyle: string;
  language: string;
  about: string;
  rules: Rule[];
}

export const loadPreferences = (): Promise<Preferences> => request<Preferences>('GET', '/api/preferences');

export const savePreferences = (preferences: Preferences): Promise<Preferences> => request<Preferences>('PUT', '/api/preferences', preferences);
