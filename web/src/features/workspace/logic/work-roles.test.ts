import { describe, expect, it } from 'vitest';
import { workRoleSegments } from './work-roles';
import type { TaskEvent, WorkSummary } from './task-types';

const work: WorkSummary = {
  id: 'work-1', updatedAt: '', state: 'open', actor: 'ChatGPT', revision: 4, planRev: 1, stale: false,
};

function event(id: string, kind: TaskEvent['kind'], text: string, extra: object = {}): TaskEvent {
  return { id, seq: Number(id), time: '', kind, text, work: { id: work.id }, ...extra } as TaskEvent;
}

describe('workRoleSegments', () => {
  it('keeps repeated roles as separate segments and derives evidence per segment', () => {
    const events: TaskEvent[] = [
      event('1', 'request', 'Start', { work: { id: work.id, plannedRoles: ['developer', 'qa', 'designer'] } }),
      event('2', 'role', 'Developer', { work: { id: work.id, role: 'developer', roleName: 'Developer' } }),
      event('3', 'action', 'Edited files', { files: ['src/a.ts'], work: { id: work.id, role: 'developer', evidence: 'observed' } }),
      event('4', 'role', 'QA', { work: { id: work.id, role: 'qa', roleName: 'QA' } }),
      event('5', 'action', 'Ran check', { run: { name: 'check', exitCode: 0 }, work: { id: work.id, role: 'qa', evidence: 'ran' } }),
      event('6', 'role', 'Developer', { work: { id: work.id, role: 'developer', roleName: 'Developer' } }),
      event('7', 'message', 'I finished', { work: { id: work.id, evidence: 'claimed' } }),
    ];

    expect(workRoleSegments(events, work)).toEqual([
      expect.objectContaining({ name: 'Developer', state: 'done', steps: [expect.objectContaining({ id: '3' })], files: ['src/a.ts'] }),
      expect.objectContaining({ name: 'QA', state: 'done', runs: [0] }),
      expect.objectContaining({ name: 'Developer', state: 'active', reports: ['I finished'] }),
      expect.objectContaining({ name: 'Designer', state: 'pending' }),
    ]);
  });

  it('does not invent roles for legacy external events without structured updates', () => {
    expect(workRoleSegments([event('1', 'message', 'Working')], work)).toEqual([]);
  });
});
