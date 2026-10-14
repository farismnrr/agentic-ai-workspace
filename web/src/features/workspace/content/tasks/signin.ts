import type { Task } from '../task-types';

export const signIn: Task = {
  id: 'sign-in',
  projectId: 'bakery',
  title: 'Fix the sign-in page',
  summary: 'Tested like a real visitor',
  status: 'review',
  updated: '10 min ago',
  started: 'Today, 9:12',
  duration: '6 min',
  model: 'Claude Sonnet',
  plan: [
    { id: 'p1', title: 'Understood your request', detail: "The sign-in button wasn't sending people anywhere.", state: 'done' },
    { id: 'p2', title: 'Made the fix', detail: 'Reconnected the button to your sign-in page.', state: 'done' },
    { id: 'p3', title: 'Tested like a real visitor', detail: 'Signed in 3 times with a test account.', state: 'done' },
    { id: 'p4', title: 'Final check', detail: 'Nothing else on your site was affected.', state: 'done' },
  ],
  events: [
    { id: 'e1', time: '9:12', kind: 'request', text: "My website's sign-in page doesn't work anymore. Can you fix it?" },
    { id: 'e2', time: '9:12', kind: 'message', text: "On it. I'll look at the sign-in page first, then fix it and test it." },
    { id: 'e3', time: '9:14', kind: 'action', text: 'Opened your sign-in page', technical: 'GET /sign-in → button#signin has no click handler' },
    { id: 'e4', time: '9:16', kind: 'action', text: 'Reconnected the button to the sign-in form', technical: 'Edited src/pages/signin.html (1 line)' },
    { id: 'e5', time: '9:19', kind: 'action', text: 'Signed in 3 times with a test account', technical: 'Ran 3 browser sessions, all redirected to /account' },
    { id: 'e6', time: '9:21', kind: 'receipt', text: 'Sign-in works. Nothing else on your site changed.' },
  ],
  changes: [{ id: 'c1', text: 'Sign-in button now sends people to your account page', files: ['src/pages/signin.html'] }],
  proof: {
    before: 'Something went wrong',
    after: 'Welcome back!',
    rows: [
      { label: 'What changed', value: 'Sign-in button fixed' },
      { label: 'How it was tested', value: '3 real sign-ins' },
      { label: 'Result', value: 'Working' },
    ],
    checks: ['Signed in with a test account', 'Wrong password shows a clear message', 'Other pages load as before'],
  },
};
