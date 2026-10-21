import { request } from '../../../shared/api/http';

export type SSLMode = 'disable' | 'allow' | 'prefer' | 'require' | 'verify-ca' | 'verify-full';
export type TLSChoice = 'none' | 'ca' | 'full';

export function tlsChoiceFromSettings(mode: SSLMode, clientCertSet: boolean): TLSChoice {
  if (mode === 'disable') return 'none';
  return clientCertSet ? 'full' : 'ca';
}

export function sslModeForTLSChoice(choice: TLSChoice): SSLMode {
  return choice === 'none' ? 'disable' : 'verify-full';
}

export interface SharedDatabaseSettings {
  host: string;
  port: number;
  database: string;
  user: string;
  sslMode: SSLMode;
  passwordSet: boolean;
  configured: boolean;
  caSet: boolean;
  clientCertSet: boolean;
  clientKeySet: boolean;
}

export interface SharedDatabaseInput extends Omit<SharedDatabaseSettings,
  'passwordSet' | 'configured' | 'caSet' | 'clientCertSet' | 'clientKeySet'> {
  password: string;
  caCert: string;
  clientCert: string;
  clientKey: string;
  removeCaCert?: boolean;
  removeClientCert?: boolean;
}

export const getSharedDatabaseSettings = (): Promise<SharedDatabaseSettings> => request('GET', '/api/settings/database');
export const saveSharedDatabaseSettings = (input: SharedDatabaseInput): Promise<SharedDatabaseSettings> =>
  request('PUT', '/api/settings/database', input);
