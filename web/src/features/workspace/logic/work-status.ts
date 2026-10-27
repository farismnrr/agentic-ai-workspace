import type { TaskStatus, WorkSummary } from './task-types';

export interface WorkStatusPresentation {
  iconStatus: TaskStatus;
  label: string;
}

export interface PlanHandoffPresentation {
  state: 'approval_saved' | 'pending' | 'complete' | 'failed';
  label: string;
}

/** Separates the saved plan decision from delivery of its handoff. */
export function planHandoffPresentation(decision?: string, handoffState?: string): PlanHandoffPresentation {
  if (decision !== 'approved') return { state: 'approval_saved', label: '' };
  if (handoffState === 'pending' || handoffState === 'created') {
    return { state: 'pending', label: 'Approval saved. The handoff is being prepared.' };
  }
  if (handoffState === 'done') {
    return { state: 'complete', label: 'Handed to the Masih Awam agent team.' };
  }
  if (handoffState === 'failed') {
    return { state: 'failed', label: 'Approval saved, but the handoff needs attention.' };
  }
  return { state: 'approval_saved', label: 'Plan approved. Handoff status is not available yet.' };
}

/** Keeps external plan decisions readable without adding another task status. */
export function workStatusPresentation(status: TaskStatus, work?: WorkSummary): WorkStatusPresentation {
  const planStatus = planStatusPresentation(work);
  if (planStatus !== undefined) return planStatus;
  if (work?.decision === 'changes_requested') {
    return { iconStatus: status, label: 'Changes requested — back with the Masih Awam agent team' };
  }
  return { iconStatus: status, label: '' };
}

function planStatusPresentation(work?: WorkSummary): WorkStatusPresentation | undefined {
  if (work?.planState === 'ready_for_approval') {
    return { iconStatus: 'needs-input', label: 'Ready for approval' };
  }
  if (work?.planState === 'approved') {
    const handoff = planHandoffPresentation(work.decision ?? 'approved', work.handoffState);
    return { iconStatus: handoff.state === 'complete' ? 'done' : 'needs-input', label: handoff.label };
  }
  if (work?.planState === 'handed_off') return { iconStatus: 'done', label: 'Handed off' };
  return undefined;
}
