import { request } from '../../../shared/api/http';

export const databaseKinds = ['postgres', 'mysql', 'mongodb'] as const;
export type DatabaseKind = typeof databaseKinds[number];
export type TLSChoice = 'none' | 'ca' | 'full';
export type CertificateField = 'ca' | 'clientCert' | 'clientKey';
export type Removal = 'ca' | 'client';

export const databaseKindLabels: Record<DatabaseKind, string> = {
  postgres: 'PostgreSQL', mysql: 'MySQL', mongodb: 'MongoDB',
};
export const databaseDefaultPorts: Record<DatabaseKind, number> = {
  postgres: 5432, mysql: 3306, mongodb: 27017,
};
export const databaseNameLabels: Record<DatabaseKind, string> = {
  postgres: 'Database name', mysql: 'Database name', mongodb: 'Auth database (usually admin)',
};

export function tlsChoiceFromSettings(tls: string, clientCertSet: boolean): TLSChoice {
  if (tls === 'none') return 'none';
  return tls === 'full' || clientCertSet ? 'full' : 'ca';
}

export function tlsForChoice(choice: TLSChoice): TLSChoice { return choice; }

export interface DatabaseSettings {
  kind: DatabaseKind;
  host: string;
  port: number;
  database: string;
  user: string;
  tls: TLSChoice;
  passwordSet: boolean;
  configured: boolean;
  caSet: boolean;
  clientCertSet: boolean;
  clientKeySet: boolean;
}

export interface DatabaseKindAvailability { kind: DatabaseKind; configured: boolean; admin: boolean }

export interface DatabaseSettingsInput {
  host: string;
  port: number;
  database: string;
  user: string;
  password: string;
  tls: TLSChoice;
  caCert: string;
  clientCert: string;
  clientKey: string;
  removeCaCert?: boolean;
  removeClientCert?: boolean;
}

export interface CertificateState {
  caSet: boolean; clientCertSet: boolean; clientKeySet: boolean;
  caCert: string; clientCert: string; clientKey: string;
  caError: string; clientCertError: string; clientKeyError: string;
}

export function certificateErrors(choice: TLSChoice, state: CertificateState): Pick<CertificateState, 'caError' | 'clientCertError' | 'clientKeyError'> {
  const caError = requiredCertificateError(choice !== 'none', state.caSet, state.caCert, state.caError, 'Choose a CA certificate.');
  const clientCertError = requiredCertificateError(choice === 'full', state.clientCertSet, state.clientCert, state.clientCertError, 'Choose a client certificate.');
  const clientKeyError = requiredCertificateError(choice === 'full', state.clientKeySet, state.clientKey, state.clientKeyError, 'Choose a client key.');
  return { caError, clientCertError, clientKeyError };
}

function requiredCertificateError(enabled: boolean, saved: boolean, value: string, error: string, message: string): string {
  if (!enabled) return '';
  if (error !== '') return error;
  return saved || value !== '' ? '' : message;
}

export const getDatabaseSettings = (kind: DatabaseKind): Promise<DatabaseSettings> =>
  request('GET', `/api/settings/database/${kind}`);
export const saveDatabaseSettings = (kind: DatabaseKind, input: DatabaseSettingsInput): Promise<DatabaseSettings> =>
  request('PUT', `/api/settings/database/${kind}`, input);
export const getDatabaseKinds = (): Promise<DatabaseKindAvailability[]> => request('GET', '/api/databases/kinds');
