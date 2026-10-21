import type { DatabaseType, EnvVariable } from './env-api';
import { ApiError } from '../../../shared/api/http';

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
