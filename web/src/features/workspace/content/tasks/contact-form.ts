import type { Task } from '../task-types';

export const contactForm: Task = {
  id: 'contact-form',
  projectId: 'bakery',
  title: 'Add a contact form',
  summary: 'Waiting for your OK to send a test email',
  status: 'working',
  updated: 'Just now',
  started: 'Today, 9:40',
  model: 'Claude Sonnet',
  plan: [
    { id: 'p1', title: 'Understanding the request', state: 'done' },
    { id: 'p2', title: 'Making the change', detail: 'Added a form with name, email and message.', state: 'done' },
    { id: 'p3', title: 'Testing it', detail: 'Needs to send one test email to check it arrives.', state: 'active' },
    { id: 'p4', title: 'Final check', state: 'pending' },
  ],
  events: [
    { id: 'e1', time: '9:40', kind: 'request', text: 'Add a contact form to my homepage so customers can message me.' },
    { id: 'e2', time: '9:41', kind: 'message', text: "Plan: add a simple form, send messages to your email, then test it. I'll ask before sending anything." },
    { id: 'e3', time: '9:44', kind: 'action', text: 'Added the form below your menu section', technical: 'Created src/components/ContactForm.html' },
    {
      id: 'e4',
      time: '9:47',
      kind: 'approval',
      text: 'To test the form I need to send one email.',
      action: 'Send 1 test email to hello@sweetcrumbs.id',
      risk: 'medium',
    },
  ],
  changes: [{ id: 'c1', text: 'New contact form on the homepage', files: ['src/components/ContactForm.html', 'src/pages/index.html'] }],
};
