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
  seq: number;
  /** ISO timestamp. */
  time: string;
}

export interface RunInfo {
  name: string;
  exitCode: number;
  timedOut?: boolean;
}

export type TaskEvent =
  | (EventBase & { kind: 'request'; text: string })
  | (EventBase & { kind: 'message'; text: string })
  | (EventBase & {
      kind: 'action';
      text: string;
      technical?: string;
      files?: string[];
      canUndo: boolean;
      undone?: boolean;
      run?: RunInfo;
    })
  | (EventBase & { kind: 'question'; text: string; options?: string[]; answered?: string })
  | (EventBase & { kind: 'approval'; text: string; action: string; risk: Risk; resolved?: 'allowed' | 'denied' })
  | (EventBase & { kind: 'error'; text: string; hint?: string })
  | (EventBase & { kind: 'receipt'; text: string })
  | (EventBase & { kind: 'site'; text: string; url: string });

export type ActionEvent = Extract<TaskEvent, { kind: 'action' }>;

export interface ProofRow {
  label: string;
  value: string;
}

/** Verified checks come from tasks the server ran itself; the rest are only claimed by the agent. */
export interface Check {
  text: string;
  verified: boolean;
  passed: boolean;
}

export interface Proof {
  rows: ProofRow[];
  checks: Check[];
  proven: boolean;
}

export interface Task {
  id: string;
  projectId: string;
  title: string;
  summary: string;
  status: TaskStatus;
  providerId: string;
  model: string;
  plan: PlanStep[];
  proof?: Proof;
  createdAt: string;
  updatedAt: string;
}

export interface Project {
  id: string;
  name: string;
  description: string;
  createdAt: string;
}

/** A change is a step of the agent that touched files. */
export interface Change {
  id: string;
  text: string;
  files: string[];
  canUndo: boolean;
  undone: boolean;
}
