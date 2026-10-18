import { request } from '../../../shared/api/http';

export type FileStatus = 'added' | 'modified' | 'deleted' | 'renamed';

/** A file the task changed: what it held before the task and what it holds now. */
export interface FileChange {
  path: string;
  oldPath?: string;
  status: FileStatus;
  before: string;
  after: string;
  /** Binary files carry no text. */
  binary: boolean;
  /** The text is cut short because the file is large. */
  truncated: boolean;
}

export const loadFileChanges = (taskId: string): Promise<FileChange[]> => request<FileChange[]>('GET', `/api/tasks/${encodeURIComponent(taskId)}/changes`);
