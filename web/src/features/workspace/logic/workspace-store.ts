import { writable } from 'svelte/store';
import { loadProjects } from './projects-api';
import { loadExtensions, type Extension } from './extensions-api';
import { loadSites, type Site } from './sites-api';
import { loadTasks, pinTask, unpinTask } from './tasks-api';
import type { Project, Task } from './task-types';

export const projects = writable<Project[]>([]);
export const tasks = writable<Task[]>([]);
export const sites = writable<Site[]>([]);
export const extensions = writable<Extension[]>([]);
export const loaded = writable(false);
/** The project the next task goes to; set when someone starts a task from a project page. */
export const chosenProject = writable('');

/** Changes a task's pin and patches the shared list with the server's canonical response. */
export async function setTaskPinned(id: string, pinned: boolean): Promise<void> {
  const updated = pinned ? await pinTask(id) : await unpinTask(id);
  tasks.update((items) => items.map((task) => (task.id === updated.id ? updated : task)));
}

const fallback = 30_000;
const debounce = 150;

/** Reloads projects and tasks; failures keep what is already shown. */
export async function refresh(): Promise<void> {
  const [projectResult, taskResult] = await Promise.allSettled([loadProjects(), loadTasks()]);
  if (projectResult.status === 'fulfilled') projects.set(projectResult.value);
  if (taskResult.status === 'fulfilled') tasks.set(taskResult.value);
  loaded.set(true);

  const [siteResult, extensionResult] = await Promise.allSettled([loadSites(), loadExtensions()]);
  if (siteResult.status === 'fulfilled') sites.set(siteResult.value);
  if (extensionResult.status === 'fulfilled') extensions.set(extensionResult.value);
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
