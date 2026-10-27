import { describe, expect, it } from 'vitest';
import { eventProvenance, isExternalReceipt, journeySteps } from './provenance';
import type { TaskEvent, WorkSummary } from './task-types';

const base = { id: 'event', seq: 1, version: 1, time: '2026-10-27T00:00:00Z' };
const external = { id: 'work', actor: 'Gemini', role: 'planner', roleName: 'Planner' };

describe('event provenance', () => {
  it.each([
    [{ ...base, kind: 'request', text: 'Request', work: external }, 'Recorded by Masih Awam'],
    [{ ...base, kind: 'message', text: 'Progress', work: external }, 'Reported by Gemini'],
    [{ ...base, kind: 'action', text: 'Step', work: external }, 'Reported by Gemini'],
    [{ ...base, kind: 'question', text: 'Question' }, 'Recorded by Masih Awam'],
    [{ ...base, kind: 'approval', text: 'Approval', action: 'write', risk: 'low' }, 'Recorded by Masih Awam'],
    [{ ...base, kind: 'error', text: 'Error' }, 'Recorded by Masih Awam'],
    [{ ...base, kind: 'receipt', text: 'Receipt' }, 'Recorded by Masih Awam'],
    [{ ...base, kind: 'site', text: 'Site', url: 'https://example.test' }, 'Recorded by Masih Awam'],
    [{ ...base, kind: 'extension', text: 'Extension', url: 'https://example.test' }, 'Recorded by Masih Awam'],
    [{ ...base, kind: 'shot', text: 'Screenshot', image: 'shot.png' }, 'Recorded by Masih Awam'],
    [{ ...base, kind: 'plan', text: 'Plan', technical: '', work: external }, 'Reported by Gemini'],
    [{ ...base, kind: 'prd', text: 'Requirements', technical: '', work: external }, 'Reported by Gemini'],
    [{ ...base, kind: 'research', text: 'Research', technical: '', work: external }, 'Reported by Gemini'],
    [{ ...base, kind: 'role', text: 'Planner', work: external }, 'Reported by Gemini'],
    [{ ...base, kind: 'handoff', text: 'Handoff', action: 'assigned', work: external }, 'Reported by Gemini'],
  ] as [TaskEvent, string][])('%s is labelled by its recorded source', (event, label) => {
    expect(eventProvenance(event).label).toBe(label);
  });

  it('keeps provider and agent role distinct', () => {
    const event = { ...base, kind: 'research', text: 'Research', technical: '', work: external } as TaskEvent;
    expect(eventProvenance(event)).toMatchObject({ provider: 'Gemini', role: 'Planner' });
  });

  it('treats an external completion as a report, never a proven receipt', () => {
    const receipt = { ...base, kind: 'receipt', text: 'Completed', work: external } as TaskEvent;
    expect(eventProvenance(receipt).label).toBe('Reported by Gemini');
    expect(isExternalReceipt(receipt)).toBe(true);
    expect(isExternalReceipt({ ...base, kind: 'receipt', text: 'Completed' })).toBe(false);
  });

  it('distinguishes recorded, approved, handed off and server-observed events', () => {
    const recorded = { ...base, kind: 'question', text: 'Which option?' } as TaskEvent;
    const approved = { ...base, kind: 'plan_review', text: 'Ready', planReview: { revision: 2, summary: 'Plan', status: 'approved' } } as TaskEvent;
    const handoff = { ...base, kind: 'plan_review', text: 'Ready', planReview: { revision: 2, summary: 'Plan', status: 'approved', linkedTaskId: 'internal' } } as TaskEvent;
    const observed = { ...base, kind: 'action', text: 'Changed a file', work: { ...external, evidence: 'observed' } } as TaskEvent;
    expect(eventProvenance(recorded).label).toBe('Recorded by Masih Awam');
    expect(eventProvenance(approved).label).toBe('Approved by you');
    expect(eventProvenance(handoff).label).toBe('Handed to the Orchestrator');
    expect(eventProvenance(handoff, '', 'queued').label).toBe('Handed to the Orchestrator');
    expect(eventProvenance(handoff, '', 'working').label).toBe('Executed by the Orchestrator');
    expect(eventProvenance(handoff, '', 'failed').label).toBe('Executed by the Orchestrator');
    expect(eventProvenance(observed).label).toBe('Verified by the server');
  });
});

describe('planning journey', () => {
  it('keeps a completed Planner role out of the proven receipt state', () => {
    const planner: TaskEvent = { ...base, kind: 'role', text: 'Planner', work: external };
    const work = { id: 'work', updatedAt: '', state: 'completed', actor: 'Gemini', revision: 1, planRev: 1, stale: false } as WorkSummary;
    expect(journeySteps([planner], work)).toEqual(['Planner']);
    expect(eventProvenance(planner).label).toBe('Reported by Gemini');
  });

  it('requires linked task execution before showing the Implementation milestone', () => {
    const events: TaskEvent[] = [
      { ...base, kind: 'role', text: 'Researcher', work: { ...external, role: 'researcher' } },
      { ...base, id: 'planner', kind: 'role', text: 'Planner', work: external },
      { ...base, id: 'question', kind: 'question', text: 'Which option?' },
    ];
    const work = { id: 'work', updatedAt: '', state: 'open', actor: 'Gemini', revision: 1, planRev: 1,
      stale: false, planState: 'ready_for_approval', linkedTaskId: 'internal' } as WorkSummary;
    expect(journeySteps(events, work)).toEqual([
      'Researcher', 'Planner', 'Needs your answer', 'Ready for approval', 'Handed off',
    ]);
    expect(journeySteps(events, work, 'queued')).not.toContain('Implementation');
    expect(journeySteps(events, work, 'working')).toContain('Implementation');
  });
});
