export type PlanReviewStatus = 'submitted' | 'approved' | 'changes_requested' | 'superseded';

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
