import { tasks } from './tasks';
import type { Task } from './task-types';

export type DayGroup = 'Today' | 'Yesterday' | 'Earlier';

export interface RecentGroup {
  label: DayGroup;
  tasks: Task[];
}

const dayOf = (task: Task): DayGroup => {
  if (/ago|just now|today/i.test(task.updated)) return 'Today';
  return task.updated === 'Yesterday' ? 'Yesterday' : 'Earlier';
};

/** Tasks grouped by day for the sidebar. `tasks` is ordered newest first. */
export function recentGroups(limit: number): RecentGroup[] {
  const groups: RecentGroup[] = [];
  for (const task of tasks.slice(0, limit)) {
    const label = dayOf(task);
    const group = groups.find((item) => item.label === label);
    if (group === undefined) groups.push({ label, tasks: [task] });
    else group.tasks.push(task);
  }
  return groups;
}
