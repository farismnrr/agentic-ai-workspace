import type { TaskEvent, TaskStatus, WorkSummary } from './task-types';

export interface Provenance {
  label: string;
  provider: string;
  role: string;
}

export type JourneyStep =
  | 'Researcher'
  | 'Planner'
  | 'Needs your answer'
  | 'Ready for approval'
  | 'Approved'
  | 'Handed off'
  | 'Implementation';

const assistantEvents = new Set<TaskEvent['kind']>([
  'message', 'action', 'research', 'plan', 'prd', 'role', 'handoff', 'receipt',
]);

/** External completion is a report, so it never receives internal receipt styling. */
export function isExternalReceipt(event: TaskEvent): boolean {
  return event.kind === 'receipt' && event.work !== undefined;
}

function isApproved(event: TaskEvent): boolean {
  return event.kind === 'plan_review' && event.planReview.status === 'approved';
}

function isHandedOff(event: TaskEvent): boolean {
  return event.kind === 'plan_review' && event.planReview.linkedTaskId !== undefined;
}

/** Describes who reported an event while keeping provider and role as separate fields. */
export function eventProvenance(event: TaskEvent, provider = '', linkedTaskStatus?: TaskStatus): Provenance {
  return {
    label: eventLabel(event, event.work?.actor ?? provider, linkedTaskStatus),
    provider: isAttributed(event) ? event.work?.actor ?? provider : '',
    role: eventRole(event),
  };
}

function eventRole(event: TaskEvent): string {
  if (!isAttributed(event)) return '';
  return event.work?.roleName ?? event.work?.role ?? event.agent ?? '';
}

function isAttributed(event: TaskEvent): boolean {
  return event.work !== undefined && (assistantEvents.has(event.kind) || isServerEvidence(event));
}

function isServerEvidence(event: TaskEvent): boolean {
  return hasServerEvidence(event);
}

function eventLabel(event: TaskEvent, actor: string, linkedTaskStatus?: TaskStatus): string {
  const handoff = handoffLabel(event, linkedTaskStatus);
  if (handoff !== undefined) return handoff;
  if (isApproved(event)) return 'Approved by you';
  if (hasServerEvidence(event)) return 'Verified by the server';
  const report = reportedLabel(event, actor);
  if (report !== undefined) return report;
  return 'Recorded by Masih Awam';
}

function handoffLabel(event: TaskEvent, linkedTaskStatus?: TaskStatus): string | undefined {
  if (!isHandedOff(event)) return undefined;
  return linkedTaskStatus !== undefined && linkedTaskStatus !== 'queued'
    ? 'Executed by the Orchestrator'
    : 'Handed to the Orchestrator';
}

function hasServerEvidence(event: TaskEvent): boolean {
  return event.work?.evidence === 'observed' || event.work?.evidence === 'ran';
}

function reportedLabel(event: TaskEvent, actor: string): string | undefined {
  if (event.work === undefined || !assistantEvents.has(event.kind)) return undefined;
  return `Reported by ${actor || 'the external assistant'}`;
}

function hasRole(roles: string[], part: string): boolean {
  return roles.some((role) => role.includes(part));
}

/** Returns only journey milestones supported by saved work and plan state. */
export function journeySteps(
  events: TaskEvent[],
  work?: WorkSummary,
  linkedTaskStatus?: TaskStatus,
): JourneyStep[] {
  const roles = events.filter((event) => event.kind === 'role')
    .map((event) => event.work?.role?.toLowerCase() ?? event.text.toLowerCase());
  const candidates: [boolean, JourneyStep][] = [
    [hasRole(roles, 'research'), 'Researcher'],
    [hasRole(roles, 'plan'), 'Planner'],
    [events.some((event) => event.kind === 'question' && !event.answered), 'Needs your answer'],
    [work?.planState === 'ready_for_approval', 'Ready for approval'],
    [work?.decision === 'approved' || events.some(isApproved), 'Approved'],
    [work?.linkedTaskId !== undefined || events.some(isHandedOff), 'Handed off'],
    [work?.linkedTaskId !== undefined && linkedTaskStatus !== undefined && linkedTaskStatus !== 'queued', 'Implementation'],
  ];
  return candidates.filter((entry) => entry[0]).map(([, step]) => step);
}
