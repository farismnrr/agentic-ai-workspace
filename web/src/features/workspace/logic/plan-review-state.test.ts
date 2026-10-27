import { describe, expect, it } from 'vitest';
import { canDecidePlan, planReviewDisplay, planReviewLabel } from './plan-review-state';

describe('plan review state', () => {
  it('allows a decision only for an editable submitted revision', () => {
    expect(canDecidePlan('submitted', false, false)).toBe(true);
    expect(canDecidePlan('submitted', true, false)).toBe(false);
    expect(canDecidePlan('submitted', false, true)).toBe(false);
    expect(canDecidePlan('approved', false, false)).toBe(false);
  });

  it('uses plain labels for each resolved plan state', () => {
    expect(planReviewLabel('submitted')).toBe('Ready for approval');
    expect(planReviewLabel('approved')).toBe('Approved');
    expect(planReviewLabel('changes_requested')).toBe('Changes requested');
    expect(planReviewLabel('superseded')).toBe('Plan changed');
  });

  it('uses a local decision only until the stream reports a later review state', () => {
    const local = {
      decision: 'approved',
      source: 'app',
      handoffState: 'created',
      linkedTaskId: 'build-task',
    };
    const waiting = planReviewDisplay({ status: 'submitted' }, local);
    expect(waiting).toMatchObject({ status: 'approved', handoffState: 'created', linkedTaskId: 'build-task' });

    const failed = planReviewDisplay({
      status: 'approved',
      source: 'app',
      handoffState: 'failed',
    }, local);
    expect(failed).toEqual({ status: 'approved', source: 'app', handoffState: 'failed' });

    const complete = planReviewDisplay({
      status: 'approved',
      source: 'app',
      handoffState: 'done',
      linkedTaskId: 'build-task',
    }, local);
    expect(complete.handoffState).toBe('done');
    expect(planReviewDisplay({ status: 'approved' }, local)).toEqual({ status: 'approved' });
  });
});
