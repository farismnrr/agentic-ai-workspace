import { get, writable, type Readable } from 'svelte/store';
import { listFiles, type FileEntry } from './files-api';

export interface FileStore {
  /** The entries of every folder read so far, by folder path ("." is the top). */
  dirs: Readable<Record<string, FileEntry[]>>;
  /** The folders that are open in the tree. */
  expanded: Readable<Set<string>>;
  toggle: (path: string) => Promise<void>;
  /** Reads the top and every open folder again, for example because the agent changed files. */
  reload: () => Promise<void>;
}

/** The tree of one project: folders are read when first opened, and again whenever the files change. */
export function createFileStore(projectId: string): FileStore {
  const dirs = writable<Record<string, FileEntry[]>>({});
  const expanded = writable(new Set<string>(['.']));

  // Reads can finish out of order when the files change in a burst; only the newest read of a folder counts.
  const latest = new Map<string, number>();
  let reads = 0;

  const load = async (path: string): Promise<void> => {
    reads += 1;
    const mine = reads;
    latest.set(path, mine);
    try {
      const listing = await listFiles(projectId, path);
      if (latest.get(path) === mine) dirs.update((all) => ({ ...all, [path]: listing.entries }));
    } catch {
      // A folder that is gone closes; anything else keeps what it showed.
      expanded.update((open) => new Set([...open].filter((entry) => entry !== path)));
    }
  };

  const toggle = async (path: string): Promise<void> => {
    const open = new Set(get(expanded));
    if (open.delete(path)) {
      expanded.set(open);
      return;
    }
    open.add(path);
    expanded.set(open);
    if (get(dirs)[path] === undefined) await load(path);
  };

  const reload = async (): Promise<void> => {
    await Promise.all([...get(expanded)].map(load));
  };

  void load('.');

  return { dirs, expanded, toggle, reload };
}
