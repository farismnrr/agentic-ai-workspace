import { request } from '../../../shared/api/http';
import type { Task, TaskEvent } from './task-types';

const base = (id: string): string => `/api/tasks/${encodeURIComponent(id)}`;

export interface NewTask {
  projectId: string;
  prompt: string;
  /** Leave out to use the default model. */
  providerId?: string;
  model?: string;
  /** Leave out to use the default effort. */
  effort?: string;
  /** Whether a lead hands the work to the AI team; leave out to follow the person's setting. */
  team?: boolean;
}

export interface UsageRow {
  providerId: string;
  model: string;
  tasks: number;
  inputTokens: number;
  outputTokens: number;
}

export const loadTasks = (): Promise<Task[]> => request<Task[]>('GET', '/api/tasks');

export const loadTask = (id: string): Promise<Task> => request<Task>('GET', base(id));

export const loadEvents = (id: string, after: number): Promise<TaskEvent[]> =>
  request<TaskEvent[]>('GET', `${base(id)}/events?after=${String(after)}`);

export const createTask = (input: NewTask): Promise<Task> => request<Task>('POST', '/api/tasks', input);

export const sendMessage = (id: string, text: string): Promise<undefined> => request<undefined>('POST', `${base(id)}/messages`, { text });

export const answerApproval = (id: string, eventId: string, allow: boolean): Promise<undefined> =>
  request<undefined>('POST', `${base(id)}/events/${encodeURIComponent(eventId)}/answer`, { allow });

export const answerQuestion = (id: string, eventId: string, choice: string): Promise<undefined> =>
  request<undefined>('POST', `${base(id)}/events/${encodeURIComponent(eventId)}/answer`, { choice });

export const undoStep = (id: string, eventId: string): Promise<undefined> =>
  request<undefined>('POST', `${base(id)}/events/${encodeURIComponent(eventId)}/undo`);

export const cancelTask = (id: string): Promise<undefined> => request<undefined>('POST', `${base(id)}/cancel`);

export const acceptTask = (id: string): Promise<undefined> => request<undefined>('POST', `${base(id)}/accept`);

export const retryTask = (id: string): Promise<undefined> => request<undefined>('POST', `${base(id)}/retry`);

export const loadUsage = (): Promise<{ since: string; models: UsageRow[] }> =>
  request<{ since: string; models: UsageRow[] }>('GET', '/api/usage');
