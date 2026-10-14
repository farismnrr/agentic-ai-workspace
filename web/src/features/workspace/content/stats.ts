import { inboxItems } from './inbox';
import { statusMeta } from './status';
import { tasks, tasksOf } from './tasks';
import type { Task, TaskStatus } from './task-types';

export type Counts = Record<TaskStatus, number>;

const empty = (): Counts => ({ queued: 0, planning: 0, working: 0, 'needs-input': 0, review: 0, done: 0, failed: 0 });

export function countByStatus(list: Task[]): Counts {
  const counts = empty();
  for (const task of list) counts[task.status] += 1;
  return counts;
}

export const projectCounts = (projectId: string): Counts => countByStatus(tasksOf(projectId));

/** How many tasks in a project wait on the user. */
export const needsYouCount = (projectId: string): number =>
  tasksOf(projectId).filter((task) => statusMeta[task.status].needsYou).length;

export const inboxCount = (): number => inboxItems().length;

export const runningTasks = (): Task[] => tasks.filter((task) => task.status === 'working' || task.status === 'planning');
