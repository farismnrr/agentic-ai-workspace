import { expect, it } from 'vitest';
import { statusMeta } from './status';
import type { TaskStatus } from '../logic/task-types';

it('gives every task status its own icon and accessible label', () => {
  const statuses: TaskStatus[] = [
    'queued', 'planning', 'working', 'needs-input', 'review', 'done', 'failed',
  ];
  const icons = statuses.map((status) => statusMeta[status].icon);
  const labels = statuses.map((status) => statusMeta[status].label);

  expect(new Set(icons).size).toBe(statuses.length);
  expect(new Set(labels).size).toBe(statuses.length);
  expect(labels.every((label) => label.trim() !== '')).toBe(true);
});
