import type { TaskEvent, WorkSummary } from './task-types';

/** External task metadata selects the external team view; otherwise show internal handoffs. */
export const hasExternalTeam = (source: string, work?: WorkSummary): boolean => work !== undefined || source !== '';

export type JobState = 'working' | 'waiting' | 'done' | 'failed';

/** One job the lead handed to a teammate, with what the thread shows they did. */
export interface TeamJob {
  /** The id of the event that handed the job over. */
  id: string;
  agent: string;
  model: string;
  brief: string;
  state: JobState;
  report: string;
  steps: number;
  files: string[];
  checks: number;
}

const emptyJob = (id: string, agent: string, model: string, brief: string): TeamJob => ({ id, agent, model, brief, state: 'working', report: '', steps: 0, files: [], checks: 0 });

/** What a step the teammate did adds to their job. */
function count(job: TeamJob, event: TaskEvent): void {
  if (event.kind === 'action') {
    job.steps += 1;
    for (const file of event.files ?? []) if (!job.files.includes(file)) job.files.push(file);
    if (event.run !== undefined) job.checks += 1;
  } else if (event.kind === 'approval') job.state = event.resolved === undefined ? 'waiting' : 'working';
}

/** A handoff back closes the teammate's open job with how it ended and what they reported. */
function close(open: Map<string, TeamJob>, agent: string, state: JobState, report: string): void {
  const job = open.get(agent);
  if (job !== undefined) Object.assign(job, { state, report });
  open.delete(agent);
}

function applyHandoff(
  open: Map<string, TeamJob>,
  jobs: TeamJob[],
  event: Extract<TaskEvent, { kind: 'handoff' }>,
  key: string,
  agent: string
): void {
  if (event.action === 'assigned') {
    const job = emptyJob(event.id, agent, event.model ?? '', event.technical ?? '');
    jobs.push(job);
    open.set(key, job);
    return;
  }
  close(open, key, event.action === 'reported' ? 'done' : 'failed', event.technical ?? '');
}

/**
 * Who did what, worked out from the thread: a handoff opens a job, the steps and checks the teammate does
 * count towards it, and a handoff back closes it. Each job is keyed by its delegation job id (falling back
 * to the teammate name for older events), so parallel jobs of the same role stay distinct.
 */
export function teamJobs(events: TaskEvent[]): TeamJob[] {
  const jobs: TeamJob[] = [];
  const open = new Map<string, TeamJob>();
  for (const event of events) {
    const agent = event.agent;
    if (agent === undefined) continue;
    const key = event.job ?? agent;
    if (event.kind === 'handoff') {
      applyHandoff(open, jobs, event, key, agent);
      continue;
    }
    const job = open.get(key) ?? open.get(agent);
    if (job !== undefined) count(job, event);
  }
  return jobs;
}
