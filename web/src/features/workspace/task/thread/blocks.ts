import type { TaskEvent } from '../../content/task-types';

export type ActionEvent = Extract<TaskEvent, { kind: 'action' }>;

export type Block = { kind: 'actions'; id: string; items: ActionEvent[] } | { kind: 'event'; event: TaskEvent };

/** Consecutive actions collapse into one expandable line; everything else stays as is. */
export function toBlocks(events: TaskEvent[]): Block[] {
  const blocks: Block[] = [];
  for (const event of events) {
    const last = blocks.at(-1);
    if (event.kind !== 'action') blocks.push({ kind: 'event', event });
    else if (last?.kind === 'actions') last.items.push(event);
    else blocks.push({ kind: 'actions', id: event.id, items: [event] });
  }
  return blocks;
}
