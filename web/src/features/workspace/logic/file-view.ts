import { writable, type Readable } from 'svelte/store';
import { ApiError } from '../../../shared/api/http';
import { deleteFile, readFile, saveFile, type OpenFile } from './files-api';

export interface ViewState {
  path: string | undefined;
  /** The file as last read from disk. */
  file: OpenFile | undefined;
  /** What the editor holds: the file plus the person's unsaved edits. */
  draft: string;
  loading: boolean;
  saving: boolean;
  /** The file changed on disk while the person has unsaved edits. */
  diverged: boolean;
  error: string;
}

export interface FileView {
  state: Readable<ViewState>;
  open: (path: string) => Promise<void>;
  edit: (text: string) => void;
  save: () => Promise<void>;
  /** The files may have changed: read the open one again. */
  refresh: () => Promise<void>;
  /** Throws away the edits and shows what is on disk now. */
  loadTheirs: () => void;
  /** Keeps the edits and makes the next save replace what is on disk. */
  keepMine: () => void;
  remove: () => Promise<void>;
  close: () => void;
}

const empty: ViewState = { path: undefined, file: undefined, draft: '', loading: false, saving: false, diverged: false, error: '' };

const failure = (error: unknown): string => (error instanceof ApiError && error.code === 'not_found' ? 'This file is gone.' : 'Something went wrong. Try again.');

interface Outcome {
  change: Partial<ViewState>;
  latest?: OpenFile | undefined;
}

/** Saves the draft against the hash it was opened with; a file that moved on is reported, not overwritten. */
async function saved(projectId: string, current: ViewState): Promise<Outcome> {
  const { path, file, draft } = current;
  if (path === undefined || file === undefined) return { change: {} };
  try {
    const { hash } = await saveFile(projectId, path, draft, file.hash);
    return { change: { file: { ...file, content: draft, hash }, saving: false, diverged: false } };
  } catch (error) {
    if (!(error instanceof ApiError && error.code === 'changed')) return { change: { saving: false, error: failure(error) } };
    return { change: { saving: false, diverged: true }, latest: await readFile(projectId, path).catch(() => undefined) };
  }
}

/** Reads the file again: a clean editor follows the disk, one with unsaved edits is marked as diverged. */
async function reread(projectId: string, current: ViewState): Promise<Outcome> {
  const { path, file, draft } = current;
  if (path === undefined || file === undefined) return { change: {} };
  try {
    const next = await readFile(projectId, path);
    if (next.hash === file.hash) return { change: {} };
    return draft === file.content ? { change: { file: next, draft: next.content } } : { change: { diverged: true }, latest: next };
  } catch (error) {
    return { change: { error: failure(error) } };
  }
}

/** The one open file of a project: what is on disk, what the person typed, and how they meet when the agent edits. */
export function createFileView(projectId: string): FileView {
  const state = writable<ViewState>(empty);
  let current: ViewState = empty;
  let latest: OpenFile | undefined;
  const set = (change: Partial<ViewState>): void => {
    current = { ...current, ...change };
    state.set(current);
  };
  const apply = (outcome: Outcome): void => {
    if (outcome.latest !== undefined) latest = outcome.latest;
    set(outcome.change);
  };

  const open = async (path: string): Promise<void> => {
    latest = undefined;
    set({ ...empty, path, loading: true });
    try {
      const file = await readFile(projectId, path);
      set({ file, draft: file.content, loading: false });
    } catch (error) {
      set({ loading: false, error: failure(error) });
    }
  };

  return {
    state, open,
    edit: (text) => { set({ draft: text }); },
    save: async () => {
      if (current.file === undefined || current.file.binary || current.file.truncated) return;
      set({ saving: true, error: '' });
      apply(await saved(projectId, current));
    },
    refresh: async () => {
      if (!current.saving) apply(await reread(projectId, current));
    },
    loadTheirs: () => {
      if (latest !== undefined) set({ file: latest, draft: latest.content, diverged: false });
    },
    keepMine: () => {
      if (latest !== undefined) set({ file: latest, diverged: false });
    },
    remove: async () => {
      if (current.path === undefined) return;
      try {
        await deleteFile(projectId, current.path);
        set(empty);
      } catch (error) {
        set({ error: failure(error) });
      }
    },
    close: () => { set(empty); },
  };
}
