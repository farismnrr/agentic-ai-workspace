import { describe, expect, it } from 'vitest';
import { planHandoffPresentation, workStatusPresentation } from './work-status';
import type { TaskStatus, WorkSummary } from './task-types';

const work = (overrides: Partial<WorkSummary> = {}): WorkSummary => ({
  id: 'work',
  updatedAt: '',
  state: 'open',
  actor: 'Gemini',
  revision: 1,
  planRev: 1,
  stale: false,
  ...overrides,
});

describe('work status presentation', () => {
  it('shows ready plans with the needs-input icon category', () => {
    expect(workStatusPresentation('needs-input', work({ planState: 'ready_for_approval' })))
      .toEqual({ iconStatus: 'needs-input', label: 'Ready for approval' });
  });

  it('distinguishes approval from each handoff state', () => {
    expect(planHandoffPresentation('approved')).toEqual({
      state: 'approval_saved', label: 'Plan approved. Handoff status is not available yet.',
    });
    for (const state of ['created', 'pending']) {
      expect(planHandoffPresentation('approved', state)).toEqual({
        state: 'pending', label: 'Approval saved. The handoff is being prepared.',
      });
    }
    expect(planHandoffPresentation('approved', 'done')).toEqual({
      state: 'complete', label: 'Handed to the Masih Awam agent team.',
    });
    expect(planHandoffPresentation('approved', 'handed_off').state).not.toBe('complete');
    expect(planHandoffPresentation('approved', 'failed')).toEqual({
      state: 'failed', label: 'Approval saved, but the handoff needs attention.',
    });
    expect(workStatusPresentation('needs-input', work({ planState: 'approved', handoffState: 'pending' })))
      .toEqual({ iconStatus: 'needs-input', label: 'Approval saved. The handoff is being prepared.' });
    expect(workStatusPresentation('done', work({ planState: 'approved', handoffState: 'done' })))
      .toEqual({ iconStatus: 'done', label: 'Handed to the Masih Awam agent team.' });
    expect(workStatusPresentation('done', work({ planState: 'handed_off' })))
      .toEqual({ iconStatus: 'done', label: 'Handed off' });
  });

  it('makes requested changes read as back with the team', () => {
    expect(workStatusPresentation('working', work({ decision: 'changes_requested' }))).toEqual({
      iconStatus: 'working',
      label: 'Changes requested — back with the Masih Awam agent team',
    });
  });

  it('leaves ordinary task statuses to the existing status metadata', () => {
    const statuses: TaskStatus[] = ['queued', 'planning', 'working', 'needs-input', 'review', 'done', 'failed'];
    for (const status of statuses) {
      expect(workStatusPresentation(status)).toEqual({ iconStatus: status, label: '' });
    }
  });
});
