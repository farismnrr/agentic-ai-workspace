import { describe, expect, it } from 'vitest';
import { ApiError } from '../../../shared/api/http';
import { describeTaskError } from './task-errors';

describe('describeTaskError', () => {
  it('explains how to recover when the Orchestrator team is unavailable', () => {
    expect(describeTaskError(new ApiError(409, 'team_unavailable'))).toBe(
      'Orchestrator is unavailable because the team is not ready. ' +
      'Ask your workspace owner to check the team setup, then try again.',
    );
  });
});
