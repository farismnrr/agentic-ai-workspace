export type TaskStatus = 'queued' | 'planning' | 'working' | 'needs-input' | 'review' | 'done' | 'failed';

export type Risk = 'low' | 'medium' | 'high';

export type PlanState = 'done' | 'active' | 'pending' | 'failed';

export interface PlanStep {
  id: string;
  title: string;
  detail?: string;
  state: PlanState;
}

interface EventBase {
  id: string;
  time: string;
}

export type TaskEvent =
  | (EventBase & { kind: 'request'; text: string })
  | (EventBase & { kind: 'message'; text: string })
  | (EventBase & { kind: 'action'; text: string; technical?: string })
  | (EventBase & { kind: 'question'; text: string; options: string[]; answered?: string })
  | (EventBase & { kind: 'approval'; text: string; action: string; risk: Risk; resolved?: 'allowed' | 'denied' })
  | (EventBase & { kind: 'error'; text: string; hint?: string })
  | (EventBase & { kind: 'receipt'; text: string });

export interface Change {
  id: string;
  text: string;
  files: string[];
}

export interface ProofRow {
  label: string;
  value: string;
}

export interface Proof {
  before: string;
  after: string;
  rows: ProofRow[];
  checks: string[];
}

export interface Task {
  id: string;
  projectId: string;
  title: string;
  summary: string;
  status: TaskStatus;
  updated: string;
  started: string;
  duration?: string;
  model: string;
  plan: PlanStep[];
  events: TaskEvent[];
  changes: Change[];
  proof?: Proof;
}
