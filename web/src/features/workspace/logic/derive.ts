import { statusMeta } from '../content/status';
import type { Change, Project, Task, TaskEvent, TaskStatus } from './task-types';

export type InboxKind = 'waiting' | 'review' | 'failed';

export interface InboxItem {
  id: string;
  kind: InboxKind;
  task: Task;
  headline: string;
}

const headlines: Record<InboxKind, string> = {
  waiting: 'Waiting for your answer',
  review: 'Finished and ready to check',
  failed: "Couldn't finish",
};

const kinds: Partial<Record<TaskStatus, InboxKind>> = { 'needs-input': 'waiting', review: 'review', failed: 'failed' };
const rank: Record<InboxKind, number> = { waiting: 0, review: 1, failed: 2 };

/** The inbox is whatever tasks are waiting on the person, so nothing is stored twice. */
export function inboxItems(list: Task[]): InboxItem[] {
  const items: InboxItem[] = [];
  for (const task of list) {
    const kind = kinds[task.status];
    if (kind !== undefined) items.push({ id: task.id, kind, task, headline: headlines[kind] });
  }
  return items.sort((a, b) => rank[a.kind] - rank[b.kind]);
}

export type Counts = Record<TaskStatus, number>;

export function countByStatus(list: Task[]): Counts {
  const counts: Counts = { queued: 0, planning: 0, working: 0, 'needs-input': 0, review: 0, done: 0, failed: 0 };
  for (const task of list) counts[task.status] += 1;
  return counts;
}

export const tasksOf = (list: Task[], projectId: string): Task[] => list.filter((task) => task.projectId === projectId);

export const needsYouCount = (list: Task[], projectId: string): number =>
  tasksOf(list, projectId).filter((task) => statusMeta[task.status].needsYou).length;

export const runningTasks = (list: Task[]): Task[] => list.filter((task) => task.status === 'working' || task.status === 'planning');

export const findProject = (list: Project[], id: string): Project | undefined => list.find((project) => project.id === id);

/** Steps of the thread that touched files, newest last. */
export function changesOf(events: TaskEvent[]): Change[] {
  const changes: Change[] = [];
  for (const event of events) {
    if (event.kind === 'action' && (event.files?.length ?? 0) > 0) {
      changes.push({
        id: event.id, text: event.text, files: event.files ?? [], canUndo: event.canUndo,
        undone: event.undone ?? false,
        ...(event.work === undefined ? {} : { work: event.work }),
        ...(event.run === undefined ? {} : { exitCode: event.run.exitCode }),
      });
    }
  }
  return changes;
}

type Of<Kind extends TaskEvent['kind']> = Extract<TaskEvent, { kind: Kind }>;

function lastOfKind<Kind extends TaskEvent['kind']>(events: TaskEvent[], kind: Kind, open: (event: Of<Kind>) => boolean): Of<Kind> | undefined {
  for (let i = events.length - 1; i >= 0; i--) {
    const event = events[i];
    if (event?.kind === kind && open(event as Of<Kind>)) return event as Of<Kind>;
  }
  return undefined;
}

/** The approval the agent is waiting on, if any. */
export const pendingApproval = (events: TaskEvent[]): Of<'approval'> | undefined => lastOfKind(events, 'approval', (event) => event.resolved === undefined);

/** The question the agent is waiting on, if any. */
export const pendingQuestion = (events: TaskEvent[]): Of<'question'> | undefined => lastOfKind(events, 'question', (event) => event.answered === undefined);

export const lastProblem = (events: TaskEvent[]): Of<'error'> | undefined => lastOfKind(events, 'error', () => true);
