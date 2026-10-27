import { request } from '../../../shared/api/http';
import type { PlanStep } from './task-types';

export interface PlanRevisionSnapshot {
  revision: number;
  summary: string;
  steps: PlanStep[];
}

export interface PlanDecisionResponse {
  decision: string;
  source?: string;
  decidedAt?: string;
  handoffId?: string;
  handoffState?: string;
  linkedTaskId?: string;
}

export const loadPlanRevision = (taskId: string, revision: number): Promise<PlanRevisionSnapshot> =>
	request<PlanRevisionSnapshot>('GET', `/api/tasks/${encodeURIComponent(taskId)}/plan/revisions/${String(revision)}`);

export const approvePlan = (
  taskId: string,
  revision: number,
  idempotencyKey: string,
): Promise<PlanDecisionResponse> => request<PlanDecisionResponse>(
  'POST', `/api/tasks/${encodeURIComponent(taskId)}/plan/approve`, { revision, idempotencyKey },
);

export const requestPlanChanges = (
  taskId: string,
  revision: number,
  idempotencyKey: string,
  note: string,
): Promise<PlanDecisionResponse> => request<PlanDecisionResponse>(
  'POST', `/api/tasks/${encodeURIComponent(taskId)}/plan/request-changes`,
  { revision, idempotencyKey, note },
);
