import type { Rule } from './types';

export const userRules: Rule[] = [
  { id: 'prices', text: 'Never change prices without asking me', on: true },
  { id: 'indonesian', text: 'Always write in Bahasa Indonesia', on: true },
  { id: 'checkout', text: "Don't touch the checkout page", on: true },
];

export const usageByModel = [
  { name: 'Claude Sonnet', tasks: 5, share: 70 },
  { name: 'GPT', tasks: 2, share: 30 },
];
