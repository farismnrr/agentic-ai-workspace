import { get } from 'svelte/store';
import { beforeEach, expect, it, vi } from 'vitest';
import type { Task, TaskEvent } from './task-types';

const { loadEvents, loadTask } = vi.hoisted(() => ({
  loadEvents: vi.fn(),
  loadTask: vi.fn(),
}));

vi.mock('./tasks-api', () => ({ loadEvents, loadTask }));

class FakeEventSource {
  static readonly CLOSED = 2;
  static readonly instances: FakeEventSource[] = [];
  readonly listeners = new Map<string, (event: MessageEvent<string>) => void>();
  readyState = 1;
  closed = false;
  onerror: (() => void) | null = null;

  constructor(readonly url: string) {
    FakeEventSource.instances.push(this);
  }

  addEventListener(type: string, listener: (event: MessageEvent<string>) => void): void {
    this.listeners.set(type, listener);
  }

  close(): void {
    this.closed = true;
    this.readyState = FakeEventSource.CLOSED;
  }

  send(type: string, data: unknown): void {
    this.listeners.get(type)?.({ data: JSON.stringify(data) } as MessageEvent<string>);
  }
}

const task = (status: Task['status'] = 'working'): Task => ({ id: 'task-1', status } as Task);
const action = (version: number, text: string, phase: string): TaskEvent => ({
  id: 'event-1', seq: 2, version, time: '2026-10-27T00:00:00Z', kind: 'action', text,
  work: { id: 'work-1', phase }, canUndo: false,
});

beforeEach(() => {
  vi.clearAllMocks();
  FakeEventSource.instances.length = 0;
  vi.stubGlobal('EventSource', FakeEventSource);
});

it('replaces a running action with its finished version without duplicates', async () => {
  const { createThread } = await import('./thread');
  const thread = createThread('task-1');
  const stream = FakeEventSource.instances[0];
  if (stream === undefined) throw new Error('thread stream was not created');
  stream.send('update', { task: task(), events: [action(2, 'Running: check', 'running')] });
  stream.send('update', { task: task(), events: [action(3, 'check finished', 'finished')] });
  stream.send('update', { task: task(), events: [action(3, 'check finished', 'finished')] });

  expect(get(thread.events)).toHaveLength(1);
  expect(get(thread.events)[0]).toMatchObject({ text: 'check finished', work: { phase: 'finished' } });
  thread.stop();
});

it('merges a changed event replayed after reconnect by id and version', async () => {
  const { createThread } = await import('./thread');
  const thread = createThread('task-1');
  const stream = FakeEventSource.instances[0];
  if (stream === undefined) throw new Error('thread stream was not created');
  stream.send('update', { task: task(), events: [action(2, 'Running: check', 'running')] });

  // EventSource reconnect replay carries the same event id with a newer version.
  stream.send('update', { task: task(), events: [action(3, 'check finished', 'finished')] });

  expect(get(thread.events)).toHaveLength(1);
  expect(get(thread.events)[0]?.version).toBe(3);
  thread.stop();
});

it('keeps live text when event replay updates an existing event during reconnect', async () => {
  const { createThread } = await import('./thread');
  const thread = createThread('task-1');
  const stream = FakeEventSource.instances[0];
  if (stream === undefined) throw new Error('thread stream was not created');
  stream.send('delta', { text: 'The result is still being written.' });
  stream.send('update', { task: task(), events: [action(2, 'Running: check', 'running')] });

  // Reconnect replays the same event with its newer persisted version.
  stream.send('update', { task: task(), events: [action(3, 'check finished', 'finished')] });

  expect(get(thread.draft)).toBe('The result is still being written.');
  expect(get(thread.events)).toHaveLength(1);
  expect(get(thread.events)[0]?.version).toBe(3);
  thread.stop();
});

it('ignores duplicate and out-of-order event updates while keeping sequence order', async () => {
  const { createThread } = await import('./thread');
  const thread = createThread('task-1');
  const stream = FakeEventSource.instances[0];
  if (stream === undefined) throw new Error('thread stream was not created');
  const later = { ...action(4, 'second', 'finished'), id: 'event-2', seq: 3 };
  stream.send('update', { task: task(), events: [action(2, 'first', 'running'), later] });
  stream.send('update', { task: task(), events: [action(3, 'first updated', 'finished'), action(1, 'old replay', 'running')] });
  stream.send('update', { task: task(), events: [action(3, 'duplicate', 'finished')] });

  expect(get(thread.events).map((event) => [event.id, event.version, event.text])).toEqual([
    ['event-1', 3, 'first updated'], ['event-2', 4, 'second'],
  ]);
  thread.stop();
});

it('keeps one current plan review card when duplicate or late versions arrive', async () => {
	const { createThread } = await import('./thread');
	const thread = createThread('task-1');
	const stream = FakeEventSource.instances[0];
	if (stream === undefined) throw new Error('thread stream was not created');
	const review = (version: number, status: string): TaskEvent => ({
		id: 'review-1', seq: 4, version, time: '2026-10-27T00:00:00Z', kind: 'plan_review',
		text: 'Plan ready for approval',
		planReview: { revision: 2, summary: 'Build the page', status },
	});
	stream.send('update', { task: task('needs-input'), events: [review(2, 'submitted')] });
	stream.send('update', { task: task('working'), events: [review(3, 'superseded')] });
	stream.send('update', { task: task('working'), events: [review(2, 'submitted')] });

	expect(get(thread.events)).toHaveLength(1);
	expect(get(thread.events)[0]).toMatchObject({ version: 3, planReview: { status: 'superseded' } });
	thread.stop();
});

it('closes the old stream and ignores a late refresh after switching tasks', async () => {
  let resolveTask: ((value: Task) => void) | undefined;
  loadTask.mockImplementation(() => new Promise<Task>((resolve) => { resolveTask = resolve; }));
  loadEvents.mockResolvedValue([]);
  const { createThread } = await import('./thread');
  const oldThread = createThread('task-1');
  const pending = oldThread.refresh();
  oldThread.stop();
  const nextThread = createThread('task-2');

  resolveTask?.(task());
  await pending;

  expect(FakeEventSource.instances[0]?.closed).toBe(true);
  expect(get(oldThread.task)).toBeUndefined();
  expect(get(oldThread.events)).toEqual([]);
  nextThread.stop();
});

it('keeps the in-memory event list bounded', async () => {
  const { createThread } = await import('./thread');
  const thread = createThread('task-1');
  const events = Array.from({ length: 2001 }, (_, index) => ({
    ...action(index + 1, `step ${String(index)}`, 'finished'), id: `event-${String(index)}`, seq: index + 1,
  }));
  FakeEventSource.instances[0]?.send('update', { task: task(), events });

  expect(get(thread.events)).toHaveLength(2000);
  expect(get(thread.events)[0]?.seq).toBe(2);
  thread.stop();
});
