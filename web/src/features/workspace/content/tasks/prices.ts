import type { Task } from '../task-types';

export const prices: Task = {
  id: 'price-update',
  projectId: 'shop',
  title: 'Update batik prices',
  summary: 'Asking which price list to use',
  status: 'needs-input',
  updated: '25 min ago',
  started: 'Today, 8:30',
  model: 'GPT',
  plan: [
    { id: 'p1', title: 'Read your price list', state: 'done' },
    { id: 'p2', title: 'Update the product pages', detail: 'Waiting for your answer first.', state: 'active' },
    { id: 'p3', title: 'Check every price on the site', state: 'pending' },
  ],
  events: [
    { id: 'e1', time: '8:30', kind: 'request', text: 'Update my batik prices using the new list I uploaded.' },
    { id: 'e2', time: '8:33', kind: 'action', text: 'Read your price list (48 products)', technical: 'Parsed prices-oct.xlsx' },
    {
      id: 'e3',
      time: '8:34',
      kind: 'question',
      text: 'Your list has two columns, "Retail" and "Reseller". Which one should the website show?',
      options: ['Retail', 'Reseller', 'Show both'],
    },
  ],
  changes: [],
};
