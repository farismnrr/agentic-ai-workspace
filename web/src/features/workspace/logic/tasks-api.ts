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

export interface TaskConfirmation {
  outcome: string;
  verified: { proven: boolean };
  already?: boolean;
}

export const loadTasks = (): Promise<Task[]> => request<Task[]>('GET', '/api/tasks');

export const loadArchivedTasks = (): Promise<Task[]> => request<Task[]>('GET', '/api/tasks/archived');

export type TaskBatchAction = 'archive' | 'restore' | 'delete';

export interface TaskBatchResult {
  id: string;
  ok: boolean;
  error?: string;
}

export async function changeTasks(action: TaskBatchAction, ids: string[]): Promise<TaskBatchResult[]> {
  const results: TaskBatchResult[] = [];
  for (let start = 0; start < ids.length; start += 1000) {
    const chunk = ids.slice(start, start + 1000);
    results.push(...await request<TaskBatchResult[]>('POST', `/api/tasks/${action}`, { ids: chunk }));
  }
  return results;
}

export const loadTask = (id: string): Promise<Task> => request<Task>('GET', base(id));

export const pinTask = (id: string): Promise<Task> => request<Task>('PUT', `${base(id)}/pin`);

export const unpinTask = (id: string): Promise<Task> => request<Task>('DELETE', `${base(id)}/pin`);

export const loadEvents = (id: string, after: number): Promise<TaskEvent[]> =>
  request<TaskEvent[]>('GET', `${base(id)}/events?after=${String(after)}`);

export const createTask = (input: NewTask): Promise<Task> => request<Task>('POST', '/api/tasks', input);

export const sendMessage = (id: string, text: string): Promise<undefined> => request<undefined>('POST', `${base(id)}/messages`, { text });

export const chatOnExternalTask = (id: string, text: string): Promise<{ linkedTaskId: string }> =>
  request<{ linkedTaskId: string }>('POST', `${base(id)}/chat`, { text });

export const answerApproval = (id: string, eventId: string, allow: boolean): Promise<undefined> =>
  request<undefined>('POST', `${base(id)}/events/${encodeURIComponent(eventId)}/answer`, { allow });

/** One list of replies per question of the form, in order. */
export const answerQuestion = (id: string, eventId: string, answers: string[][]): Promise<undefined> =>
  request<undefined>('POST', `${base(id)}/events/${encodeURIComponent(eventId)}/answer`, { answers });

export const undoStep = (id: string, eventId: string): Promise<undefined> =>
  request<undefined>('POST', `${base(id)}/events/${encodeURIComponent(eventId)}/undo`);

export const cancelTask = (id: string): Promise<undefined> => request<undefined>('POST', `${base(id)}/cancel`);

export const confirmTask = (id: string): Promise<TaskConfirmation> =>
  request<TaskConfirmation>('POST', `${base(id)}/confirm`);

export const acceptTask = (id: string): Promise<undefined> => request<undefined>('POST', `${base(id)}/accept`);

export const retryTask = (id: string): Promise<undefined> => request<undefined>('POST', `${base(id)}/retry`);

export const loadUsage = (): Promise<{ since: string; models: UsageRow[] }> =>
  request<{ since: string; models: UsageRow[] }>('GET', '/api/usage');

/** Where a screenshot of a task is served; the page sends the session cookie with the image request. */
export const shotUrl = (id: string, image: string): string => `${base(id)}/shots/${encodeURIComponent(image)}`;

export interface ProjectDocument {
  path: string;
  title: string;
  taskId: string;
  status: Task['status'];
  updatedAt: string;
}

export const loadDocuments = (id: string): Promise<ProjectDocument[]> =>
  request<ProjectDocument[]>('GET', `${base(id)}/documents`);
