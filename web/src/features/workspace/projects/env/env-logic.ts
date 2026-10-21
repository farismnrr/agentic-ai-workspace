import type { EnvVariable } from './env-api';
import { ApiError } from '../../../../shared/api/http';

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

const databaseProvisionMessages: Record<string, string> = {
  rate_limited: 'Too many attempts. Wait a minute and try again.',
  env_name_in_use: 'A database address already exists in Variables. Rename it before choosing this database.',
  kind_not_configured: 'This database is not set up yet. Ask an admin to add it in Settings.',
  admin_url_invalid: 'The shared database address is invalid. Contact your workspace administrator.',
  admin_not_configured: 'The shared database is not configured. Contact your workspace administrator.',
  admin_rejected: 'The shared database administrator login was rejected. Contact your workspace administrator.',
  insufficient_privileges: 'The shared database account does not have permission to create a project database.',
  provision_failed: 'Could not prepare this database. Try again.',
  provision_in_progress: 'This database is already being prepared. Wait a moment and try again.',
  compensation_failed: 'The save failed and the previous database setup could not be fully restored. Contact your workspace administrator.',
  manual_connection_exists: 'Remove the custom connection before choosing a managed database.',
};

export function databaseProvisionErrorMessage(code: string | undefined): string {
  return code === undefined
    ? 'Could not prepare this database. Try again.'
    : databaseProvisionMessages[code] ?? 'Could not prepare this database. Try again.';
}

export function databaseTestErrorMessage(code: string | undefined): string {
  if (code === 'rate_limited') return 'Too many attempts. Wait a minute and try again.';
  return 'Could not check the shared database. Try again.';
}
