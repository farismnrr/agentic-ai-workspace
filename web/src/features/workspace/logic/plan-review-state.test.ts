import { describe, expect, it } from 'vitest';
import { canDecidePlan, planReviewLabel } from './plan-review-state';

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
});
