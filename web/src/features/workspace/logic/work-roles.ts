import type { ActionEvent, TaskEvent, WorkSummary } from './task-types';

export type WorkRoleState = 'done' | 'active' | 'pending' | 'waiting';

export interface WorkRoleSegment {
  id: string;
  key: string;
  name: string;
  brief: string;
  state: WorkRoleState;
  steps: ActionEvent[];
  files: string[];
  runs: number[];
  reports: string[];
}

const label = (key: string, name: string): string => name.trim() || key;
const displayRole = (key: string): string => key.split(/[-_\s]+/).filter(Boolean)
  .map((part) => part.length <= 3
    ? part.toUpperCase()
    : `${(part[0] ?? '').toUpperCase()}${part.slice(1)}`).join(' ');

function addAction(segment: WorkRoleSegment, event: TaskEvent): void {
  if (event.kind !== 'action') return;
  segment.steps.push(event);
  if (event.run !== undefined) segment.runs.push(event.run.exitCode);
}

function addObservedFiles(segment: WorkRoleSegment, event: TaskEvent): void {
  if (event.work?.evidence !== 'observed' || event.kind !== 'action') return;
  for (const file of event.files ?? []) {
    if (!segment.files.includes(file)) segment.files.push(file);
  }
}

function addClaimedReport(segment: WorkRoleSegment, event: TaskEvent): void {
  if (event.work?.evidence !== 'claimed' || event.text.trim() === '') return;
  const technical = event.kind === 'message' ? event.technical ?? '' : '';
  segment.reports.push(technical.trim() === '' ? event.text : `${event.text}\n${technical}`);
}

function addEvidence(segment: WorkRoleSegment, event: TaskEvent): void {
  addAction(segment, event);
  addObservedFiles(segment, event);
  addClaimedReport(segment, event);
}

function roleFromEvent(event: TaskEvent): WorkRoleSegment | undefined {
  if (event.kind !== 'role' || event.work?.role === undefined) return undefined;
  return {
    id: event.id,
    key: event.work.role,
    name: label(event.work.role, event.work.roleName ?? event.text),
    brief: event.technical ?? '',
    state: 'active', steps: [], files: [], runs: [], reports: [],
  };
}

function collectSegments(events: TaskEvent[], workID: string): {
  segments: WorkRoleSegment[];
  seen: Set<string>;
} {
  const segments: WorkRoleSegment[] = [];
  const seen = new Set<string>();
  let active: WorkRoleSegment | undefined;
  for (const event of events) {
    if (event.work?.id !== workID) continue;
    const next = roleFromEvent(event);
    if (next !== undefined) {
      active = next;
      segments.push(active);
      seen.add(active.key);
    } else if (active !== undefined) addEvidence(active, event);
  }
  return { segments, seen };
}

function pendingRoles(events: TaskEvent[], seen: Set<string>): WorkRoleSegment[] {
  const begin = events.find((event) => event.work?.plannedRoles !== undefined);
  return (begin?.work?.plannedRoles ?? []).filter((role) => !seen.has(role)).map((key) => ({
    id: `pending-${key}`, key, name: displayRole(key), brief: '', state: 'pending',
    steps: [], files: [], runs: [], reports: [],
  }));
}

function finishSegments(segments: WorkRoleSegment[], work: WorkSummary): void {
  for (const segment of segments.slice(0, -1)) segment.state = 'done';
  const current = segments.at(-1);
  if (current === undefined) return;
  if (work.state !== 'open' && work.state !== 'blocked') current.state = 'done';
  else if (work.phase === 'awaiting_person' || work.planState === 'ready_for_approval') current.state = 'waiting';
}

/** Roles are grouped only from recorded role transitions and the begin_work plan. */
export function workRoleSegments(events: TaskEvent[], work: WorkSummary): WorkRoleSegment[] {
  const { segments, seen } = collectSegments(events, work.id);
  finishSegments(segments, work);
  return [...segments, ...pendingRoles(events, seen)];
}
