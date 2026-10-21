import { request } from '../../../shared/api/http';

export interface EnvVariable {
  name: string;
  isSecret: boolean;
  updatedAt: string;
  value?: string;
}

export type DatabaseType = 'postgres' | 'mysql' | 'mongodb';

export interface DatabaseConnection {
  type: DatabaseType;
  host: string;
  port: number;
  database: string;
  user: string;
  tls: boolean;
  envName: string;
  configured: boolean;
}

export interface DatabaseInput extends Omit<DatabaseConnection, 'configured'> {
  password: string;
}

export interface TestResult {
  ok: boolean;
  category?: string;
  message: string;
}

const projectPath = (id: string): string => `/api/projects/${encodeURIComponent(id)}`;

export const listVariables = (id: string): Promise<EnvVariable[]> => request('GET', `${projectPath(id)}/env`);
export const saveVariable = (id: string, item: { name: string; value: string; isSecret: boolean }): Promise<EnvVariable> =>
  request('PUT', `${projectPath(id)}/env`, item);
export const deleteVariable = (id: string, name: string): Promise<undefined> =>
  request('DELETE', `${projectPath(id)}/env/${encodeURIComponent(name)}`);

export const getDatabase = (id: string): Promise<DatabaseConnection> => request('GET', `${projectPath(id)}/db`);
export const saveDatabase = (id: string, item: DatabaseInput): Promise<DatabaseConnection> =>
  request('PUT', `${projectPath(id)}/db`, item);
export const deleteDatabase = (id: string): Promise<undefined> => request('DELETE', `${projectPath(id)}/db`);
export const testDatabase = (id: string, item?: DatabaseInput): Promise<TestResult> =>
  request('POST', `${projectPath(id)}/db/test`, item);
