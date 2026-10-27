import { describe, expect, it } from 'vitest';
import { changesOf } from './derive';
import type { TaskEvent } from './task-types';

describe('changesOf', () => {
  it('derives undo and file data from external action events', () => {
    const events: TaskEvent[] = [{
      id: 'change-1', seq: 1, version: 1, time: '', kind: 'action', text: 'Updated page',
      files: ['src/page.svelte'], canUndo: true, undone: false,
      work: { id: 'work-1', actor: 'ChatGPT', role: 'developer', evidence: 'observed' },
    }];

    expect(changesOf(events)).toEqual([{
      id: 'change-1', text: 'Updated page', files: ['src/page.svelte'],
      canUndo: true, undone: false, work: events[0]?.work,
    }]);
  });

  it('omits changes for external events without file edits', () => {
    const events: TaskEvent[] = [{
      id: 'message-1', seq: 1, version: 1, time: '', kind: 'message', text: 'Working',
      work: { id: 'work-1' },
    }];

    expect(changesOf(events)).toEqual([]);
  });
});
