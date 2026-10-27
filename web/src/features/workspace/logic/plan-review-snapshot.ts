import type { PlanRevisionSnapshot } from './plan-review-api';

export interface PlanSnapshotSection {
  label: string;
  content: string;
}

export function planReviewSnapshotSections(snapshot: PlanRevisionSnapshot): PlanSnapshotSection[] {
  return [
    { label: 'Requirements', content: snapshot.prd ?? 'No requirements were supplied.' },
    { label: 'Research', content: snapshot.research ?? 'No research was supplied.' },
    { label: 'Questions and answers', content: snapshot.qa ?? 'No questions or answers were supplied.' },
  ];
}
