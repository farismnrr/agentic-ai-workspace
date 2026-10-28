import { writable, type Readable } from 'svelte/store';
import { readGitStatus, followFiles, type GitFileStatus, type GitSnapshot } from './files-api';

export interface GitStatusStore {
  snapshot: Readable<GitSnapshot | undefined>;
  refresh: () => Promise<void>;
  stop: () => void;
}

const unavailable: GitSnapshot = { available: false, initial: false, files: [] };

/** Reads one project's current Git state, never task action history. */
export function createGitStatusStore(projectId: string): GitStatusStore {
  const snapshot = writable<GitSnapshot>();
  let revision = 0;
  let timer: ReturnType<typeof setTimeout> | undefined;
  let polling: ReturnType<typeof setInterval> | undefined;

  const refresh = async (): Promise<void> => {
    const requestId = ++revision;
    try {
      const next = await readGitStatus(projectId);
      if (requestId === revision) snapshot.set(next);
    } catch {
      if (requestId === revision) snapshot.set(unavailable);
    }
  };

  const schedule = (): void => {
    clearTimeout(timer);
    timer = setTimeout(() => void refresh(), 350);
  };
  const stopFiles = typeof window === 'undefined' ? () => undefined : followFiles(projectId, schedule);
  const onFocus = (): void => { if (document.visibilityState === 'visible') void refresh(); };
  if (typeof window !== 'undefined') {
    window.addEventListener('focus', onFocus);
    document.addEventListener('visibilitychange', onFocus);
    polling = setInterval(() => {
      if (document.visibilityState === 'visible') void refresh();
    }, 15000);
  }
  void refresh();

  return {
    snapshot,
    refresh,
    stop: () => {
      revision += 1;
      clearTimeout(timer);
      clearInterval(polling);
      stopFiles();
      if (typeof window !== 'undefined') {
        window.removeEventListener('focus', onFocus);
        document.removeEventListener('visibilitychange', onFocus);
      }
    },
  };
}

export const gitStatusByPath = (files: readonly GitFileStatus[]): Map<string, GitFileStatus> =>
  new Map(files.flatMap((file) => [[file.path, file], ...(file.originalPath ? [[file.originalPath, file]] : [])]));

export function gitStatusLabel(file: GitFileStatus): string {
  if (file.conflict) return 'Conflict';
  if (file.untracked) return 'Untracked';
  const codes = `${file.index}${file.worktree}`;
  if (codes.includes('D')) return 'Deleted';
  if (codes.includes('R') || file.originalPath) return 'Renamed';
  if (codes.includes('A')) return 'Added';
  return 'Modified';
}
