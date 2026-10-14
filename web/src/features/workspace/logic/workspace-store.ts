import { writable } from 'svelte/store';
import { loadProjects } from './projects-api';
import { loadSites, type Site } from './sites-api';
import { loadTasks } from './tasks-api';
import type { Project, Task } from './task-types';

export const projects = writable<Project[]>([]);
export const tasks = writable<Task[]>([]);
export const sites = writable<Site[]>([]);
export const loaded = writable(false);
/** The project the next task goes to; set when someone starts a task from a project page. */
export const chosenProject = writable('');

const fallback = 30_000;
const debounce = 150;

/** Reloads projects and tasks; failures keep what is already shown. */
export async function refresh(): Promise<void> {
  try {
    const [nextProjects, nextTasks, nextSites] = await Promise.all([loadProjects(), loadTasks(), loadSites()]);
    projects.set(nextProjects);
    tasks.set(nextTasks);
    sites.set(nextSites);
    loaded.set(true);
  } catch {
    // The next change or fallback poll tries again; stale data is better than an error flash.
  }
}

/**
 * Keeps the stores fresh: the server announces changes over an event stream, and a slow poll
 * repairs anything missed (for example while the connection was down). Returns a stop function.
 */
export function watch(): () => void {
  let stopped = false;
  let pending: ReturnType<typeof setTimeout> | undefined;
  let poll: ReturnType<typeof setTimeout> | undefined;

  const reload = (): void => {
    if (pending !== undefined) clearTimeout(pending);
    pending = setTimeout(() => void refresh(), debounce);
  };

  const tick = async (): Promise<void> => {
    await refresh();
    if (!stopped) poll = setTimeout(() => void tick(), fallback);
  };

  const source = new EventSource('/api/stream');
  source.addEventListener('changed', reload);
  source.addEventListener('open', reload);
  void tick();

  return () => {
    stopped = true;
    source.close();
    if (pending !== undefined) clearTimeout(pending);
    if (poll !== undefined) clearTimeout(poll);
  };
}
