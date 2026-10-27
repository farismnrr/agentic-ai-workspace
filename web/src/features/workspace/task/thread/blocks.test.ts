import { expect, it } from 'vitest';
import { toBlocks } from './blocks';
import type { TaskEvent } from '../../logic/task-types';

const message = (id: string): TaskEvent => ({
  id,
  seq: id === 'first' ? 1 : 2,
  version: 1,
  time: '2026-10-27T00:00:00Z',
  kind: 'message',
  text: 'Run the same check',
});

it('keeps repeated generated content under distinct event keys', () => {
  const blocks = toBlocks([message('first'), message('second')]);

  expect(blocks).toHaveLength(2);
  expect(blocks.map((block) => block.kind === 'actions' ? block.id : block.event.id)).toEqual(['first', 'second']);
});
