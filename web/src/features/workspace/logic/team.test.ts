import { describe, expect, it } from 'vitest';
import { hasExternalTeam } from './team';
import type { WorkSummary } from './task-types';

const work: WorkSummary = {
  id: 'work-1', updatedAt: '', state: 'open', actor: 'ChatGPT', revision: 1, planRev: 0, stale: false,
};

describe('hasExternalTeam', () => {
  it('selects external work from its source or work summary', () => {
    expect(hasExternalTeam('ChatGPT')).toBe(true);
    expect(hasExternalTeam('', work)).toBe(true);
  });

  it('keeps tasks without external metadata on the internal team view', () => {
    expect(hasExternalTeam('')).toBe(false);
  });
});
