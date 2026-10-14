import { writable, type Readable } from 'svelte/store';
import { loadEvents, loadTask } from './tasks-api';
import type { Task, TaskEvent } from './task-types';

const quick = 1200;
const slow = 5000;

export interface Thread {
  task: Readable<Task | undefined>;
  events: Readable<TaskEvent[]>;
  /** True when the task could not be loaded at all. */
  missing: Readable<boolean>;
  /** Fetches right now, for example after the person pressed a button. */
  refresh: () => Promise<void>;
  stop: () => void;
}

const active = (task: Task | undefined): boolean => task !== undefined && ['queued', 'planning', 'working'].includes(task.status);

/** Follows one task: loads it with its thread and keeps polling while the agent works. */
export function createThread(taskId: string): Thread {
  const task = writable<Task | undefined>(undefined);
  const events = writable<TaskEvent[]>([]);
  const missing = writable(false);
  let current: Task | undefined;
  let seen: TaskEvent[] = [];
  let stopped = false;
  let timer: ReturnType<typeof setTimeout> | undefined;

  const refresh = async (): Promise<void> => {
    try {
      const last = seen.at(-1)?.seq ?? 0;
      const [nextTask, fresh] = await Promise.all([loadTask(taskId), loadEvents(taskId, last)]);
      current = nextTask;
      task.set(nextTask);
      missing.set(false);
      if (fresh.length > 0) {
        seen = [...seen, ...fresh];
        events.set(seen);
      }
    } catch {
      if (current === undefined) missing.set(true);
    }
  };

  const tick = async (): Promise<void> => {
    await refresh();
    if (!stopped) timer = setTimeout(() => void tick(), active(current) ? quick : slow);
  };

  void tick();

  return {
    task,
    events,
    missing,
    refresh,
    stop: () => {
      stopped = true;
      if (timer !== undefined) clearTimeout(timer);
    },
  };
}
