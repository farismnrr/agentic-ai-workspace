import { writable, type Readable } from 'svelte/store';
import { loadFileChanges, type FileChange } from './changes-api';
import { followFiles } from './files-api';

export interface FileChanges {
  files: Readable<FileChange[]>;
  loaded: Readable<boolean>;
  /** Reads what the task changed again. */
  refresh: () => Promise<void>;
  stop: () => void;
}

/** What a task changed, file by file. It reads again whenever the project's files change, as the task works. */
export function createFileChanges(taskId: string, projectId: string): FileChanges {
  const files = writable<FileChange[]>([]);
  const loaded = writable(false);
  let reads = 0;

  const refresh = async (): Promise<void> => {
    reads += 1;
    const mine = reads;
    try {
      const next = await loadFileChanges(taskId);
      if (mine === reads) files.set(next);
    } catch {
      // Keep what is shown; the next change reads again.
    }
    loaded.set(true);
  };

  let timer: ReturnType<typeof setTimeout> | undefined;
  const stopFollowing = followFiles(projectId, () => {
    clearTimeout(timer);
    timer = setTimeout(() => { void refresh(); }, 400);
  });

  void refresh();

  return { files, loaded, refresh, stop: () => { clearTimeout(timer); stopFollowing(); } };
}
