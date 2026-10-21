import { request } from '../../../../shared/api/http';

export interface EnvVariable {
  name: string;
  isSecret: boolean;
  updatedAt: string;
  value?: string;
}

export interface DatabaseConnection {
  choice: 'sqlite' | 'postgres' | 'mysql' | 'mongodb' | 'custom';
  type: 'postgres' | 'mysql' | 'mongodb' | '';
  host: string;
  port: number;
  database: string;
  user: string;
  tls: boolean;
  envName: string;
  configured: boolean;
  managed: boolean;
  readOnly: boolean;
}

export type DatabaseKind = 'postgres' | 'mysql' | 'mongodb';
export interface DatabaseKindAvailability { kind: DatabaseKind; configured: boolean; admin: boolean }

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
export const getDatabaseKinds = (): Promise<DatabaseKindAvailability[]> => request('GET', '/api/databases/kinds');
export const detachSQLiteDatabase = (id: string): Promise<undefined> => request('DELETE', `${projectPath(id)}/db/sqlite`);
export const testDatabase = (id: string): Promise<TestResult> => request('POST', `${projectPath(id)}/db/test`);

export const provisionDatabase = (id: string, kind: DatabaseKind): Promise<DatabaseConnection> =>
  request('POST', `${projectPath(id)}/db/provision`, { kind });
