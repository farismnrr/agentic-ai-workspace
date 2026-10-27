import type { PlanStep } from './task-types';

export interface PlanReviewData {
  revision: number;
  summary: string;
  steps?: PlanStep[];
  status: string;
  decision?: string;
  source?: string;
  decidedAt?: string;
  handoffId?: string;
  handoffState?: string;
  linkedTaskId?: string;
  smallTask?: boolean;
}
