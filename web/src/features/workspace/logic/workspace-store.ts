import { writable } from 'svelte/store';
import { loadProjects } from './projects-api';
import { loadTasks } from './tasks-api';
import type { Project, Task } from './task-types';

export const projects = writable<Project[]>([]);
export const tasks = writable<Task[]>([]);
export const loaded = writable(false);
/** The project the next task goes to; set when someone starts a task from a project page. */
export const chosenProject = writable('');

const quick = 2000;
const slow = 10_000;

/** Reloads projects and tasks; failures keep what is already shown. */
export async function refresh(): Promise<void> {
  try {
    const [nextProjects, nextTasks] = await Promise.all([loadProjects(), loadTasks()]);
    projects.set(nextProjects);
    tasks.set(nextTasks);
    loaded.set(true);
  } catch {
    // The next poll tries again; stale data is better than an error flash.
  }
}

const running = (list: Task[]): boolean => list.some((task) => ['queued', 'planning', 'working'].includes(task.status));

/** Keeps the stores fresh: often while the agent is working, rarely otherwise. Returns a stop function. */
export function watch(): () => void {
  let stopped = false;
  let timer: ReturnType<typeof setTimeout> | undefined;
  let latest: Task[] = [];
  const unsubscribe = tasks.subscribe((value) => {
    latest = value;
  });

  const tick = async (): Promise<void> => {
    await refresh();
    if (!stopped) timer = setTimeout(() => void tick(), running(latest) ? quick : slow);
  };

  void tick();

  return () => {
    stopped = true;
    unsubscribe();
    if (timer !== undefined) clearTimeout(timer);
  };
}
