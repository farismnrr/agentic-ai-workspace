import { writable, type Readable, type Writable } from 'svelte/store';
import { readGitStatus, followFiles, type GitFileStatus, type GitSnapshot } from './files-api';

export interface GitStatusStore {
  snapshot: Readable<GitSnapshot | undefined>;
  refresh: () => Promise<void>;
  stop: () => void;
}

interface SharedStatus {
  snapshot: Writable<GitSnapshot | undefined>;
  refresh: () => Promise<void>;
  stop: () => void;
  users: number;
}

const unavailable: GitSnapshot = { available: false, initial: false, files: [] };
const projects = new Map<string, SharedStatus>();

/** Shares one live repository snapshot across the Explorer and task Changes panel. */
export function createGitStatusStore(projectId: string): GitStatusStore {
  let shared = projects.get(projectId);
  if (shared === undefined) {
    shared = createSharedStatus(projectId);
    projects.set(projectId, shared);
  }
  shared.users += 1;
  let active = true;
  return {
    snapshot: shared.snapshot,
    refresh: shared.refresh,
    stop: () => {
      if (!active) return;
      active = false;
      shared.users -= 1;
      if (shared.users === 0) {
        shared.stop();
        projects.delete(projectId);
      }
    },
  };
}

function createSharedStatus(projectId: string): SharedStatus {
  const snapshot = writable<GitSnapshot>();
  let revision = 0;
  let debounce: ReturnType<typeof setTimeout> | undefined;
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
    clearTimeout(debounce);
    debounce = setTimeout(() => void refresh(), 350);
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
    users: 0,
    stop: () => {
      revision += 1;
      clearTimeout(debounce);
      clearInterval(polling);
      stopFiles();
      if (typeof window !== 'undefined') {
        window.removeEventListener('focus', onFocus);
        document.removeEventListener('visibilitychange', onFocus);
      }
    },
  };
}

export function gitStatusByPath(files: readonly GitFileStatus[]): Map<string, GitFileStatus> {
  const statuses = new Map<string, GitFileStatus>();
  for (const file of files) {
    statuses.set(file.path, file);
    if (file.originalPath !== undefined) statuses.set(file.originalPath, file);
  }
  return statuses;
}

export function gitStatusLabel(file: GitFileStatus): string {
  if (file.conflict) return 'Conflict';
  if (file.untracked) return 'Untracked';
  const codes = `${file.index}${file.worktree}`;
  if (codes.includes('D')) return 'Deleted';
  if (codes.includes('R') || file.originalPath) return 'Renamed';
  if (codes.includes('A')) return 'Added';
  return 'Modified';
}
