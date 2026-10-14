import type { Task, TaskStatus } from '../task-types';

interface Simple {
  id: string;
  projectId: string;
  title: string;
  request: string;
  result: string;
  status?: TaskStatus;
  updated: string;
}

/** Builds a short mock task so projects other than the featured ones still have believable history. */
export function simple({ id, projectId, title, request, result, status = 'done', updated }: Simple): Task {
  const done = status === 'done';
  return {
    id,
    projectId,
    title,
    summary: done ? 'Accepted by you' : 'Waiting to start',
    status,
    updated,
    started: updated,
    ...(done ? { duration: '4 min' } : {}),
    model: 'Claude Sonnet',
    plan: [
      { id: 'p1', title: 'Understood your request', state: done ? 'done' : 'active' },
      { id: 'p2', title: 'Made the change', state: done ? 'done' : 'pending' },
      { id: 'p3', title: 'Checked it works', state: done ? 'done' : 'pending' },
    ],
    events: [
      { id: 'e1', time: updated, kind: 'request', text: request },
      ...(done ? [{ id: 'e2', time: updated, kind: 'receipt' as const, text: result }] : []),
    ],
    changes: done ? [{ id: 'c1', text: result, files: [] }] : [],
  };
}
