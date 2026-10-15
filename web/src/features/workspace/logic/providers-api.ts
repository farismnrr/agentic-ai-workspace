import { request } from '../../../shared/api/http';

export type ProviderKind = 'anthropic' | 'openai' | 'gemini' | 'compatible';

export interface Provider {
  id: string;
  kind: ProviderKind;
  name: string;
  baseUrl: string;
  keyHint: string;
  models: string[];
  verified: boolean;
  enabled: boolean;
  /** Tokens each model can hold at once: the size chosen, else the built-in default. */
  contextWindows: Record<string, number>;
}

export interface ModelOption {
  providerId: string;
  providerName: string;
  model: string;
}

export interface NewProvider {
  kind: ProviderKind;
  name: string;
  baseUrl: string;
  apiKey: string;
}

export const loadProviders = (): Promise<Provider[]> => request<Provider[]>('GET', '/api/providers');

export const addProvider = (input: NewProvider): Promise<Provider> => request<Provider>('POST', '/api/providers', input);

export const verifyProvider = (id: string): Promise<Provider> => request<Provider>('POST', `/api/providers/${encodeURIComponent(id)}/verify`);

export const setProviderEnabled = (id: string, enabled: boolean): Promise<undefined> =>
  request<undefined>('PUT', `/api/providers/${encodeURIComponent(id)}`, { enabled });

export const setContextWindow = (id: string, model: string, contextWindow: number): Promise<undefined> =>
  request<undefined>('PUT', `/api/providers/${encodeURIComponent(id)}/context`, { model, contextWindow });

export const removeProvider = (id: string): Promise<undefined> => request<undefined>('DELETE', `/api/providers/${encodeURIComponent(id)}`);

export const loadModels = (): Promise<ModelOption[]> => request<ModelOption[]>('GET', '/api/models');
