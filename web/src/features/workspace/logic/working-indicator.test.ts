import { expect, it } from 'vitest';
import { shouldShowWorkingIndicator } from './working-indicator';
import type { TaskStatus, WorkSummary } from './task-types';

const openWork: WorkSummary = {
  id: 'work-1', updatedAt: '', state: 'open', actor: 'assistant', revision: 1,
  planRev: 0, stale: false,
};

it('shows activity for active task statuses without terminal, blocked, or stale work', () => {
  const active: TaskStatus[] = ['queued', 'planning', 'working'];
  for (const status of active) expect(shouldShowWorkingIndicator(status)).toBe(true);
  expect(shouldShowWorkingIndicator('working', openWork)).toBe(true);
  for (const state of ['blocked', 'completed', 'failed', 'stopped', 'abandoned'] as const) {
    expect(shouldShowWorkingIndicator('working', { ...openWork, state })).toBe(false);
  }
  expect(shouldShowWorkingIndicator('working', { ...openWork, stale: true })).toBe(false);
});

it('hides activity for task statuses that are not active', () => {
  const inactive: TaskStatus[] = ['needs-input', 'review', 'done', 'failed'];
  for (const status of inactive) expect(shouldShowWorkingIndicator(status)).toBe(false);
});
