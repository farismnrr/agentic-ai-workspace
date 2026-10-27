import type { PlanRevisionSnapshot } from './plan-review-api';

export interface PlanSnapshotSection {
  label: string;
  content: string;
}

export function planReviewSnapshotSections(snapshot: PlanRevisionSnapshot): PlanSnapshotSection[] {
  return [
    { label: 'PRD', content: snapshot.prd ?? 'No PRD was supplied.' },
    { label: 'Research', content: snapshot.research ?? 'No research was supplied.' },
    { label: 'Questions and answers', content: snapshot.qa ?? 'No questions or answers were supplied.' },
  ];
}
