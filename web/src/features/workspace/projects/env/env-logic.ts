import type { DatabaseType, EnvVariable } from './env-api';
import { ApiError } from '../../../../shared/api/http';

export const databaseDefaults: Record<DatabaseType, { port: number; envName: string }> = {
  postgres: { port: 5432, envName: 'DATABASE_URL' },
  mysql: { port: 3306, envName: 'DATABASE_URL' },
  mongodb: { port: 27017, envName: 'MONGODB_URL' },
};

export function normalizeVariableName(name: string): string {
  return name.toUpperCase();
}

export function formatUpdatedAt(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? 'Unknown' : date.toLocaleString();
}

export function variableValue(variable: EnvVariable): string {
  return variable.isSecret ? '••••••' : variable.value ?? '';
}

export function variableErrorMessage(error: unknown): string {
  if (!(error instanceof ApiError)) return 'Could not save this variable. Try again.';
  switch (error.code) {
    case 'reserved_name': return 'That name is reserved for the system. Choose another name.';
    case 'invalid_name': return 'Use a valid name: start with a letter or underscore, then use uppercase letters, numbers, or underscores.';
    case 'invalid_value': return 'Enter a valid value of up to 4 KB.';
    case 'variable_limit': return 'This project has reached its 50-variable limit.';
    default: return 'Could not save this variable. Check its name and value, then try again.';
  }
}

export type DatabaseMode = 'manual' | 'shared';

const sharedProvisionMessages: Record<string, string> = {
  rate_limited: 'Too many attempts. Wait a minute and try again.',
  env_name_in_use: 'An environment variable named DATABASE_URL already exists. Delete or rename it in Variables first.',
  admin_url_invalid: 'The server shared database address is invalid. Contact your workspace administrator.',
  admin_not_configured: 'Shared Postgres is not configured. Contact your workspace administrator.',
  admin_rejected: 'The shared database administrator login was rejected. Contact your workspace administrator.',
  insufficient_privileges: 'The shared database account does not have permission to create your schema.',
  provision_failed: 'Could not create your database schema. Try again.',
  provision_in_progress: 'Your schema is already being created. Wait a moment and try again.',
  manual_connection_exists: 'A manual connection exists. Delete it first, then create the shared schema.',
  compensation_failed: 'The save failed and the previous database setup could not be fully restored. Contact your workspace administrator.',
};

export function sharedProvisionErrorMessage(code: string | undefined): string {
  return code === undefined
    ? 'Could not create your database schema. Try again.'
    : sharedProvisionMessages[code] ?? 'Could not create your database schema. Try again.';
}

export function databaseTestErrorMessage(code: string | undefined): string {
  if (code === 'rate_limited') return 'Too many attempts. Wait a minute and try again.';
  return 'Could not test this connection. Check the fields and try again.';
}
