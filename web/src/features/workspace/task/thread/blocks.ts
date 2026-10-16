import type { ActionEvent, TaskEvent } from '../../logic/task-types';

export type Block = { kind: 'actions'; id: string; items: ActionEvent[] } | { kind: 'event'; event: TaskEvent };

/** Consecutive actions by the same agent collapse into one expandable line; everything else stays as is. */
export function toBlocks(events: TaskEvent[]): Block[] {
  const blocks: Block[] = [];
  for (const event of events) {
    const last = blocks.at(-1);
    if (event.kind !== 'action') blocks.push({ kind: 'event', event });
    else if (last?.kind === 'actions' && last.items[0]?.agent === event.agent) last.items.push(event);
    else blocks.push({ kind: 'actions', id: event.id, items: [event] });
  }
  return blocks;
}
