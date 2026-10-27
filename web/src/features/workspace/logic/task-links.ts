import type { Task } from './task-types';

export function isExternalTask(source: string, workId: string): boolean {
  return source !== '' || workId !== '';
}

export function taskHref(projectId: string, taskId: string): string {
  return `/app/projects/${encodeURIComponent(projectId)}/tasks/${encodeURIComponent(taskId)}`;
}

export function originTaskHref(task: Task): string | undefined {
  return task.originTaskId === undefined ? undefined : taskHref(task.projectId, task.originTaskId);
}
