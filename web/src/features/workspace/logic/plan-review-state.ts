export type PlanReviewStatus = 'submitted' | 'approved' | 'changes_requested' | 'superseded';

export interface PlanReviewDisplay {
  status: string;
  source?: string | undefined;
  decidedAt?: string | undefined;
  handoffState?: string | undefined;
  linkedTaskId?: string | undefined;
}

type StreamedPlanReview = PlanReviewDisplay;

interface LocalPlanDecision {
  decision: string;
  source?: string | undefined;
  decidedAt?: string | undefined;
  handoffState?: string | undefined;
  linkedTaskId?: string | undefined;
}

/** Uses local approval feedback until the event stream reports the saved decision. */
export function planReviewDisplay(
  review: StreamedPlanReview,
  result?: LocalPlanDecision,
): PlanReviewDisplay {
  if (review.status !== 'submitted') return review;
  return {
    status: result?.decision ?? review.status,
    source: preferValue(result?.source, review.source),
    decidedAt: preferValue(result?.decidedAt, review.decidedAt),
    handoffState: preferValue(result?.handoffState, review.handoffState),
    linkedTaskId: preferValue(result?.linkedTaskId, review.linkedTaskId),
  };
}

function preferValue<T>(first: T | undefined, fallback: T | undefined): T | undefined {
  return first ?? fallback;
}

export function planReviewLabel(status: string): string {
  switch (status) {
    case 'submitted': return 'Ready for approval';
    case 'approved': return 'Approved';
    case 'changes_requested': return 'Changes requested';
    case 'superseded': return 'Plan changed';
    default: return 'Plan review';
  }
}

export function canDecidePlan(status: string, readOnly: boolean, inFlight: boolean): boolean {
  return status === 'submitted' && !readOnly && !inFlight;
}

export function newPlanDecisionKey(): string {
  return crypto.randomUUID();
}
