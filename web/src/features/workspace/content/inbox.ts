import { tasks } from './tasks';
import type { Task } from './task-types';

export type InboxKind = 'approval' | 'question' | 'review' | 'failed';

export interface InboxItem {
  id: string;
  kind: InboxKind;
  task: Task;
  headline: string;
}

const hasPendingApproval = (task: Task): boolean => task.events.some((event) => event.kind === 'approval' && event.resolved === undefined);

function kindOf(task: Task): InboxKind | undefined {
  if (hasPendingApproval(task)) return 'approval';
  if (task.status === 'needs-input') return 'question';
  if (task.status === 'review') return 'review';
  return task.status === 'failed' ? 'failed' : undefined;
}

const headlines: Record<InboxKind, string> = {
  approval: 'Wants your OK to continue',
  question: 'Has a question for you',
  review: 'Finished and ready to check',
  failed: "Couldn't finish",
};

export const kindLabel = headlines;

const rank: Record<InboxKind, number> = { approval: 0, question: 1, review: 2, failed: 3 };

/** Inbox items are derived from tasks, so nothing is stored twice. */
export function inboxItems(): InboxItem[] {
  const items: InboxItem[] = [];
  for (const task of tasks) {
    const kind = kindOf(task);
    if (kind !== undefined) items.push({ id: task.id, kind, task, headline: headlines[kind] });
  }
  return items.sort((a, b) => rank[a.kind] - rank[b.kind]);
}
