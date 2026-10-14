import type { Task } from '../task-types';

export const gallery: Task = {
  id: 'gallery',
  projectId: 'shop',
  title: 'Add a photo gallery',
  summary: "Couldn't finish: photos are too large",
  status: 'failed',
  updated: '2 hours ago',
  started: 'Today, 7:05',
  duration: '12 min',
  model: 'GPT',
  plan: [
    { id: 'p1', title: 'Understood your request', state: 'done' },
    { id: 'p2', title: 'Add the gallery section', state: 'done' },
    { id: 'p3', title: 'Test on a phone', detail: 'The page took 14 seconds to open.', state: 'failed' },
    { id: 'p4', title: 'Final check', state: 'pending' },
  ],
  events: [
    { id: 'e1', time: '7:05', kind: 'request', text: 'Add a photo gallery for my best batik designs.' },
    { id: 'e2', time: '7:10', kind: 'action', text: 'Added a gallery with 24 photos', technical: 'Created gallery.html, 24 images (62 MB)' },
    {
      id: 'e3',
      time: '7:17',
      kind: 'error',
      text: 'The page is too slow on phones, so I stopped before it could go live.',
      hint: 'I can shrink the photos to about 5 MB total. Want me to try that?',
    },
  ],
  changes: [{ id: 'c1', text: 'Gallery page (not published)', files: ['gallery.html'] }],
};
