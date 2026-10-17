import { request } from '../../../shared/api/http';

export interface FileEntry {
  name: string;
  path: string;
  size: number;
  dir: boolean;
}

export interface FileListing {
  path: string;
  entries: FileEntry[];
  truncated: boolean;
}

export interface OpenFile {
  path: string;
  content: string;
  /** What an edit must quote, so a save is refused when the file changed after it was opened. */
  hash: string;
  /** Only the start of a file too large to edit is shown. */
  truncated: boolean;
  binary: boolean;
}

const base = (projectId: string): string => `/api/projects/${encodeURIComponent(projectId)}`;
const query = (path: string): string => `?path=${encodeURIComponent(path)}`;

export const listFiles = (projectId: string, path: string): Promise<FileListing> => request<FileListing>('GET', `${base(projectId)}/files${query(path)}`);

export const readFile = (projectId: string, path: string): Promise<OpenFile> => request<OpenFile>('GET', `${base(projectId)}/file${query(path)}`);

/** Saves an edit; an empty baseHash creates a file that must not exist yet. Fails with code "changed" when it moved on. */
export const saveFile = (projectId: string, path: string, content: string, baseHash: string): Promise<{ hash: string }> =>
  request<{ hash: string }>('PUT', `${base(projectId)}/file`, { path, content, baseHash });

export const deleteFile = (projectId: string, path: string): Promise<undefined> => request<undefined>('DELETE', `${base(projectId)}/file${query(path)}`);

/** Calls onchange whenever the project's files may have changed (the agent worked, or another page saved). */
export function followFiles(projectId: string, onchange: () => void): () => void {
  const source = new EventSource(`${base(projectId)}/files/stream`);
  source.addEventListener('changed', onchange);
  return () => { source.close(); };
}
