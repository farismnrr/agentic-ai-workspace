import type { IconName } from '../../shared/ui/Icon.svelte';

export const REPO_URL = 'https://github.com/farismnrr/agentic-ai-workspace';

export interface NavLink {
  label: string;
  href: string;
}

export interface Reassurance {
  icon: IconName;
  label: string;
}

export interface ChecklistItem {
  label: string;
  status: 'Done' | 'In progress';
}

export const navLinks: NavLink[] = [
  { label: 'How it works', href: '#how-it-works' },
  { label: "Why it's different", href: '#why-different' },
  { label: 'FAQ', href: '#faq' },
];

export const footerLinks: NavLink[] = [...navLinks, { label: 'GitHub', href: REPO_URL }];

export const reassurances: Reassurance[] = [
  { icon: 'heart', label: 'Free & open source' },
  { icon: 'laptop', label: 'Runs on your computer' },
];

export const heroChecklist: ChecklistItem[] = [
  { label: 'Understanding the request', status: 'Done' },
  { label: 'Making the fix', status: 'Done' },
  { label: 'Testing like a real visitor', status: 'Done' },
  { label: 'Final check', status: 'Done' },
];

export const stepChecklist: ChecklistItem[] = [
  { label: 'Understanding the request', status: 'Done' },
  { label: 'Making the fix', status: 'Done' },
  { label: 'Testing like a real visitor', status: 'In progress' },
];

export interface Step {
  number: string;
  title: string;
  text: string;
}

export const steps: Step[] = [
  { number: '1', title: 'Say what you need', text: 'Type it in plain words, like texting a friend.' },
  { number: '2', title: 'Your AI team gets to work', text: 'Watch every step as it happens.' },
  { number: '3', title: 'Accept only what works', text: 'Nothing is final until you approve it.' },
];

export interface FaqEntry {
  question: string;
  answer: string;
}

export const faq: FaqEntry[] = [
  {
    question: 'Do I need to know coding?',
    answer: 'No. If you can describe what you need in plain words, your AI team can work on it.',
  },
  {
    question: 'Is it really free?',
    answer: 'Yes. Masih Awam is free and open source, with no subscription. Anyone can see how it works.',
  },
  {
    question: 'Are my files safe?',
    answer: "Your files stay on your computer. It runs there, not on someone else's server.",
  },
  {
    question: 'What if it gets something wrong?',
    answer: 'Nothing is final until you approve it. You watch each step, and you can say no.',
  },
];
