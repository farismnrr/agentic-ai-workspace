import { describe, expect, it } from 'vitest';
import { planReviewSnapshotSections } from './plan-review-snapshot';

describe('plan review snapshot sections', () => {
  it('returns the exact approved documents as plain text sections', () => {
    const sections = planReviewSnapshotSections({
      revision: 3,
      summary: 'Implement the feature',
      steps: [],
      prd: '<script>alert("text only")</script>',
      research: 'Source: https://example.test/research',
      qa: '[{"question":"Which option?","answer":"A"}]',
    });

    expect(sections).toEqual([
      { label: 'Requirements', content: '<script>alert("text only")</script>' },
      { label: 'Research', content: 'Source: https://example.test/research' },
      { label: 'Questions and answers', content: '[{"question":"Which option?","answer":"A"}]' },
    ]);
  });

  it('labels missing optional documents without omitting snapshot sections', () => {
    const sections = planReviewSnapshotSections({ revision: 1, summary: 'Small task', steps: [] });

    expect(sections.map(({ content }) => content)).toEqual([
      'No requirements were supplied.',
      'No research was supplied.',
      'No questions or answers were supplied.',
    ]);
  });
});
