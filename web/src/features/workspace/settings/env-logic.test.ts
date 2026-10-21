import { describe, expect, it } from 'vitest';
import type { EnvVariable } from './env-api';
import { databaseDefaults, formatUpdatedAt, normalizeVariableName, variableErrorMessage, variableValue } from './env-logic';
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
});
