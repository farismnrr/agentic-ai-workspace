import type { TaskStatus, WorkSummary } from './task-types';

type WorkPhase = WorkSummary['phase'];

export interface WorkStatusPresentation {
  iconStatus: TaskStatus;
  label: string;
  hint: string;
  showWorking: boolean;
  canStop: boolean;
  canConfirm: boolean;
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
  const activity = activityPresentation(status, work);
  if (isTerminalTask(status) || isTerminalWork(work) || work?.phase === 'running') {
    return activity ?? emptyPresentation(status, work);
  }

  const planStatus = planStatusPresentation(work);
  if (planStatus !== undefined) {
    return { ...planStatus, hint: '', ...taskActions(status, work), showWorking: false };
  }
  if (work?.decision === 'changes_requested') {
    return {
      iconStatus: status,
      label: 'Changes requested — back with the Masih Awam agent team',
      hint: '',
      showWorking: false,
      ...taskActions(status, work),
    };
  }
  if (activity !== undefined) return activity;
  return {
    iconStatus: status, label: '', hint: '', showWorking: canShowTaskActivity(status, work),
    ...taskActions(status, work),
  };
}

function emptyPresentation(status: TaskStatus, work?: WorkSummary): WorkStatusPresentation {
  return {
    iconStatus: status, label: '', hint: '', showWorking: false,
    ...taskActions(status, work),
  };
}

const activityCopy: Partial<Record<WorkPhase, Pick<WorkStatusPresentation, 'iconStatus' | 'label' | 'hint'>>> = {
  running: { iconStatus: 'working', label: 'Working on it', hint: '' },
  awaiting_person: { iconStatus: 'needs-input', label: 'Needs your answer', hint: 'The assistant is waiting for your answer.' },
  blocked: { iconStatus: 'needs-input', label: 'Needs your attention', hint: 'This work is blocked and needs your attention.' },
  reported: { iconStatus: 'review', label: 'Reported, not confirmed', hint: 'The assistant has given its answer.' },
  idle: {
    iconStatus: 'needs-input', label: 'No updates for over 30 min',
    hint: 'There have been no recent updates. Mark complete only if the work is finished.',
  },
};

function activityPresentation(status: TaskStatus, work?: WorkSummary): WorkStatusPresentation | undefined {
  if (isTerminalTask(status)) {
    return { iconStatus: status, label: '', hint: '', showWorking: false, canStop: false, canConfirm: false };
  }
  const copy = work?.phase === undefined ? undefined : activityCopy[work.phase];
  if (copy === undefined) {
    return isTerminalWork(work)
      ? { iconStatus: status, label: '', hint: '', showWorking: false, canStop: false, canConfirm: false }
      : undefined;
  }
  return {
    ...copy,
    showWorking: work?.phase === 'running' && canShowTaskActivity(status, work),
    ...taskActions(status, work),
  };
}

function taskActions(status: TaskStatus, work?: WorkSummary): Pick<WorkStatusPresentation, 'canStop' | 'canConfirm'> {
  if (isTerminalTask(status) || isTerminalWork(work)) return { canStop: false, canConfirm: false };
  const phase = work?.phase;
  if (phase === 'reported' || phase === 'idle') return { canStop: true, canConfirm: true };
  if (phase === 'running' || phase === 'awaiting_person' || phase === 'blocked') {
    return { canStop: true, canConfirm: false };
  }
  return { canStop: isTaskRunning(status), canConfirm: false };
}

const isTaskRunning = (status: TaskStatus): boolean => ['queued', 'planning', 'working'].includes(status);
const canShowTaskActivity = (status: TaskStatus, work?: WorkSummary): boolean =>
  !isTerminalTask(status) && !isTerminalWork(work) && (work?.phase === undefined ? isTaskRunning(status) : work.phase === 'running');
const isTerminalTask = (status: TaskStatus): boolean => ['done', 'failed'].includes(status);
const isTerminalWork = (work?: WorkSummary): boolean =>
  work?.phase === 'completed' || work?.phase === 'failed' || work?.phase === 'stopped' || work?.phase === 'abandoned';

function planStatusPresentation(work?: WorkSummary): Pick<WorkStatusPresentation, 'iconStatus' | 'label'> | undefined {
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
