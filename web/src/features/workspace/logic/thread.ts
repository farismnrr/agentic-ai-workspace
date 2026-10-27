import { writable, type Readable, type Writable } from 'svelte/store';
import { loadEvents, loadTask } from './tasks-api';
import type { Task, TaskEvent } from './task-types';

const maxThreadEvents = 2000;

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

interface EventMerger {
  cursor: () => number;
  hasWaiting: () => boolean;
  merge: (fresh: TaskEvent[]) => void;
}

/** True while a question or an approval in the thread still shows as waiting. */
const isWaiting = (list: TaskEvent[]): boolean =>
  list.some((event) => (event.kind === 'question' && event.answered === undefined) || (event.kind === 'approval' && event.resolved === undefined));

function createEventMerger(events: Writable<TaskEvent[]>, draft: Writable<string>): EventMerger {
  let seen: TaskEvent[] = [];
  let cursor = 0;
  return {
    cursor: () => cursor,
    hasWaiting: () => isWaiting(seen),
    merge: (fresh) => {
      const byId = new Map(seen.map((event) => [event.id, event]));
      let changed = false;
      for (const event of fresh) {
        cursor = Math.max(cursor, event.version);
        const prior = byId.get(event.id);
        if (prior === undefined || event.version > prior.version) {
          byId.set(event.id, event);
          changed = true;
        }
      }
      if (!changed) return;
      seen = [...byId.values()].sort((left, right) => left.seq - right.seq).slice(-maxThreadEvents);
      events.set(seen);
      draft.set('');
    },
  };
}

function listenToThread(
  source: EventSource,
  apply: (task: Task, events: TaskEvent[]) => void,
  appendDraft: (text: string) => void,
  onMissing: () => void,
  isStopped: () => boolean,
): void {
  source.addEventListener('update', (event) => {
    if (isStopped()) return;
    const payload = JSON.parse((event as MessageEvent<string>).data) as UpdatePayload;
    apply(payload.task, payload.events);
  });
  source.addEventListener('delta', (event) => {
    if (isStopped()) return;
    const { text } = JSON.parse((event as MessageEvent<string>).data) as { text: string };
    appendDraft(text);
  });
  source.onerror = (): void => {
    if (!isStopped() && source.readyState === EventSource.CLOSED) onMissing();
  };
}

/** Follows one task over a server-sent event stream: thread entries arrive as they happen. */
export function createThread(taskId: string): Thread {
  const task = writable<Task | undefined>(undefined);
  const events = writable<TaskEvent[]>([]);
  const draft = writable('');
  const missing = writable(false);
  const merger = createEventMerger(events, draft);
  let current: Task | undefined;
  let reloaded = '';
  let stopped = false;

  /** Refreshes changed events after a waiting card's task state moves on. */
  const reconcile = (status: string): void => {
    const key = `${status}:${String(merger.cursor())}`;
    if (stopped || status === 'needs-input' || !merger.hasWaiting() || reloaded === key) return;
    reloaded = key;
    void loadEvents(taskId, merger.cursor()).then((fresh) => {
      if (!stopped) merger.merge(fresh);
    }).catch(() => undefined);
  };

  const apply = (nextTask: Task, fresh: TaskEvent[]): void => {
    if (stopped) return;
    current = nextTask;
    task.set(nextTask);
    missing.set(false);
    merger.merge(fresh);
    reconcile(nextTask.status);
  };

  const source = new EventSource(`/api/tasks/${encodeURIComponent(taskId)}/stream`);
  listenToThread(source, apply, (text) => {
    draft.update((value) => value + text);
  }, () => {
    if (current === undefined) missing.set(true);
  }, () => stopped);

  const refresh = async (): Promise<void> => {
    try {
      const [nextTask, fresh] = await Promise.all([loadTask(taskId), loadEvents(taskId, merger.cursor())]);
      apply(nextTask, fresh);
    } catch {
      if (!stopped && current === undefined) missing.set(true);
    }
  };

  return {
    task,
    events,
    draft,
    missing,
    refresh,
    stop: () => {
      stopped = true;
      source.close();
    },
  };
}
