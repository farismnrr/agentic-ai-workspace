import { describe, expect, it } from 'vitest';
import type { EnvVariable } from './env-api';
import { databaseDefaults, databaseTestErrorMessage, formatUpdatedAt, normalizeVariableName, sharedProvisionErrorMessage, variableErrorMessage, variableValue } from './env-logic';
import { ApiError } from '../../../shared/api/http';

describe('environment settings logic', () => {
  it('normalizes names for the API while typing', () => {
    expect(normalizeVariableName('service_token')).toBe('SERVICE_TOKEN');
  });

  it('masks secrets and displays only non-secret values', () => {
    const secret: EnvVariable = { name: 'TOKEN', isSecret: true, updatedAt: '', value: 'hidden' };
    const visible: EnvVariable = { name: 'MODE', isSecret: false, updatedAt: '', value: 'test' };
    expect(variableValue(secret)).toBe('••••••');
    expect(variableValue(visible)).toBe('test');
  });

  it('uses the expected port and environment name for each database type', () => {
    expect(databaseDefaults.postgres).toEqual({ port: 5432, envName: 'DATABASE_URL' });
    expect(databaseDefaults.mysql).toEqual({ port: 3306, envName: 'DATABASE_URL' });
    expect(databaseDefaults.mongodb).toEqual({ port: 27017, envName: 'MONGODB_URL' });
  });

  it('formats valid dates and handles invalid dates', () => {
    expect(formatUpdatedAt('not-a-date')).toBe('Unknown');
    expect(formatUpdatedAt('2026-01-01T00:00:00.000Z')).not.toBe('Unknown');
  });

  it('turns stable variable API errors into plain messages', () => {
    expect(variableErrorMessage(new ApiError(400, 'reserved_name'))).toContain('reserved');
    expect(variableErrorMessage(new ApiError(400, 'invalid_name'))).toContain('valid name');
    expect(variableErrorMessage(new ApiError(400, 'invalid_value'))).toContain('4 KB');
    expect(variableErrorMessage(new ApiError(409, 'variable_limit'))).toContain('50-variable limit');
    expect(variableErrorMessage(new ApiError(500, 'internal'))).toContain('Could not save');
  });

  it('maps shared database provisioning failures to plain messages', () => {
    expect(sharedProvisionErrorMessage('admin_not_configured')).toContain('not configured');
    expect(sharedProvisionErrorMessage('admin_rejected')).toContain('login was rejected');
    expect(sharedProvisionErrorMessage('insufficient_privileges')).toContain('permission');
    expect(sharedProvisionErrorMessage('provision_failed')).toContain('Try again');
    expect(sharedProvisionErrorMessage('provision_in_progress')).toContain('already being created');
    expect(sharedProvisionErrorMessage('manual_connection_exists')).toBe(
      'A manual connection exists. Delete it first, then create the shared schema.',
    );
    expect(sharedProvisionErrorMessage('unknown')).toContain('Try again');
    expect(sharedProvisionErrorMessage('env_name_in_use')).toBe(
      'An environment variable named DATABASE_URL already exists. Delete or rename it in Variables first.',
    );
    expect(sharedProvisionErrorMessage('admin_url_invalid')).toContain('address is invalid');
    expect(sharedProvisionErrorMessage('rate_limited')).toBe('Too many attempts. Wait a minute and try again.');
  });

  it('shows a friendly database test rate limit message', () => {
    expect(databaseTestErrorMessage('rate_limited')).toBe('Too many attempts. Wait a minute and try again.');
    expect(databaseTestErrorMessage('other')).toContain('Could not test');
  });
});
