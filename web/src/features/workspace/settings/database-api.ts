import { request } from '../../../shared/api/http';

export interface SharedDatabaseSettings {
  host: string;
  port: number;
  database: string;
  user: string;
  sslMode: string;
  passwordSet: boolean;
  configured: boolean;
}

export interface SharedDatabaseInput extends Omit<SharedDatabaseSettings, 'passwordSet' | 'configured'> {
  password: string;
}

export const getSharedDatabaseSettings = (): Promise<SharedDatabaseSettings> => request('GET', '/api/settings/database');
export const saveSharedDatabaseSettings = (input: SharedDatabaseInput): Promise<SharedDatabaseSettings> =>
  request('PUT', '/api/settings/database', input);
