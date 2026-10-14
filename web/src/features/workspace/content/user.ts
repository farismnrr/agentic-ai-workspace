export const NOT_CONNECTED = 'This is a preview. Nothing was sent or saved yet.';

export const suggestions = ['Fix a broken button', 'Add a contact form', 'Make my site faster'];

export function greeting(hour: number): string {
  if (hour < 12) return 'Good morning';
  return hour < 18 ? 'Good afternoon' : 'Good evening';
}

export const plan = { name: 'Free plan', used: 7, limit: 20 };

export const askLevels = [
  { value: 'always', label: 'Ask me first' },
  { value: 'risky', label: 'Only for risky steps' },
];
