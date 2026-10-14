import { writable, type Readable } from 'svelte/store';
import { loadEvents, loadTask } from './tasks-api';
import type { Task, TaskEvent } from './task-types';

export interface Thread {
  task: Readable<Task | undefined>;
  events: Readable<TaskEvent[]>;
  /** Words the AI is writing right now; empty once they become a thread entry. */
  draft: Readable<string>;
  /** True when the task could not be loaded at all. */
  missing: Readable<boolean>;
  /** Fetches right now, for example after the person pressed a button. */
  refresh: () => Promise<void>;
  stop: () => void;
}

interface UpdatePayload {
  task: Task;
  events: TaskEvent[];
}

/** Follows one task over a server-sent event stream: thread entries arrive as they happen. */
export function createThread(taskId: string): Thread {
  const task = writable<Task | undefined>(undefined);
  const events = writable<TaskEvent[]>([]);
  const draft = writable('');
  const missing = writable(false);
  let current: Task | undefined;
  let seen: TaskEvent[] = [];

  const apply = (nextTask: Task, fresh: TaskEvent[]): void => {
    current = nextTask;
    task.set(nextTask);
    missing.set(false);
    const last = seen.at(-1)?.seq ?? 0;
    const added = fresh.filter((event) => event.seq > last);
    if (added.length > 0) {
      seen = [...seen, ...added];
      events.set(seen);
      draft.set('');
    }
  };

  const source = new EventSource(`/api/tasks/${encodeURIComponent(taskId)}/stream`);
  source.addEventListener('update', (event) => {
    const payload = JSON.parse((event as MessageEvent<string>).data) as UpdatePayload;
    apply(payload.task, payload.events);
  });
  source.addEventListener('delta', (event) => {
    const { text } = JSON.parse((event as MessageEvent<string>).data) as { text: string };
    draft.update((value) => value + text);
  });
  // A stream the server refuses (unknown task, signed out) is closed for good; temporary drops are retried by the browser.
  source.onerror = (): void => {
    if (source.readyState === EventSource.CLOSED && current === undefined) missing.set(true);
  };

  const refresh = async (): Promise<void> => {
    try {
      const [nextTask, fresh] = await Promise.all([loadTask(taskId), loadEvents(taskId, seen.at(-1)?.seq ?? 0)]);
      apply(nextTask, fresh);
    } catch {
      if (current === undefined) missing.set(true);
    }
  };

  return { task, events, draft, missing, refresh, stop: () => { source.close(); } };
}
