import type { IconName } from '../../../shared/ui/icon-names';
import type { TaskStatus } from '../logic/task-types';

export type StatusTone = 'primary' | 'info' | 'warning' | 'error' | 'neutral';

export interface StatusMeta {
  label: string;
  icon: IconName;
  tone: StatusTone;
  needsYou: boolean;
}

/** One place for how every task status looks: label, icon, tone, and whether it waits on the user. */
export const statusMeta: Record<TaskStatus, StatusMeta> = {
  queued: { label: 'Waiting to start', icon: 'clock', tone: 'neutral', needsYou: false },
  planning: { label: 'Making a plan', icon: 'file-text', tone: 'info', needsYou: false },
  working: { label: 'Working on it', icon: 'loader', tone: 'warning', needsYou: false },
  'needs-input': { label: 'Needs your answer', icon: 'help-circle', tone: 'warning', needsYou: true },
  review: { label: 'Ready for your review', icon: 'eye', tone: 'primary', needsYou: true },
  done: { label: 'Done', icon: 'check-circle', tone: 'neutral', needsYou: false },
  failed: { label: 'Needs attention', icon: 'alert-triangle', tone: 'error', needsYou: true },
};

export const statusOrder: TaskStatus[] = ['needs-input', 'review', 'failed', 'working', 'planning', 'queued', 'done'];
