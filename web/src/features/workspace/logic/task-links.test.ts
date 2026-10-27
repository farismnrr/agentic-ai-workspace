import { expect, it } from 'vitest';
import { isExternalTask, originTaskHref, taskHref } from './task-links';
import type { Task } from './task-types';

it('links a linked conversation back to its originating external task', () => {
  const task = { id: 'internal-1', projectId: 'project one', originTaskId: 'external/1' } as Task;

  expect(originTaskHref(task)).toBe('/app/projects/project%20one/tasks/external%2F1');
});

it('keeps a link to the linked conversation in the same project', () => {
  expect(taskHref('project-1', 'internal-1')).toBe('/app/projects/project-1/tasks/internal-1');
});

it('recognizes external work sessions with either a source or work ID', () => {
  expect(isExternalTask('ChatGPT', '')).toBe(true);
  expect(isExternalTask('', 'work-1')).toBe(true);
  expect(isExternalTask('', '')).toBe(false);
});
