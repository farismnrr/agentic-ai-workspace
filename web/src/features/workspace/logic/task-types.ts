export type TaskStatus = 'queued' | 'planning' | 'working' | 'needs-input' | 'review' | 'done' | 'failed';

export type Risk = 'low' | 'medium' | 'high';

/** Work handed to a teammate, handed back, or given up. */
export type HandoffState = 'assigned' | 'reported' | 'failed';

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
  /** The teammate who did it; absent for the lead and for tasks without a team. */
  agent?: string;
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
  | (EventBase & { kind: 'question'; text: string; options?: string[]; questions?: Question[]; answered?: string })
  | (EventBase & { kind: 'approval'; text: string; action: string; risk: Risk; resolved?: 'allowed' | 'denied' })
  | (EventBase & { kind: 'error'; text: string; hint?: string })
  | (EventBase & { kind: 'receipt'; text: string })
  | (EventBase & { kind: 'site'; text: string; url: string })
  | (EventBase & { kind: 'shot'; text: string; image: string })
  | (EventBase & { kind: 'prd'; text: string; technical: string })
  | (EventBase & { kind: 'handoff'; text: string; technical?: string; action: HandoffState; model?: string });

/** How one question of a form is answered: pick one, pick several, or type. */
export type QuestionType = 'single' | 'multi' | 'text';

export interface QuestionOption {
  label: string;
  description?: string;
}

export interface Question {
  header?: string;
  text: string;
  type: QuestionType;
  options?: QuestionOption[];
}

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
  /** The teammate who ran the check; absent for the lead and for tasks without a team. */
  by?: string;
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
  /** How hard the AI was asked to think; empty is the provider's default. */
  effort: string;
  /** A lead hands the work to the AI team. */
  team: boolean;
  /** The requirements document the team's Planner wrote, in markdown; empty when there is none. */
  prd: string;
  plan: PlanStep[];
  proof?: Proof;
  /** Tokens the last model call carried and the model's window; both 0 before the first call. */
  contextUsed: number;
  contextWindow: number;
  createdAt: string;
  updatedAt: string;
}

export interface Project {
  id: string;
  name: string;
  description: string;
  /** The connected app that owns the project, such as ChatGPT; empty for projects made by hand. */
  source: string;
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
