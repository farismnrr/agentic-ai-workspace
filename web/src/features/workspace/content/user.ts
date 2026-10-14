export const suggestions = ['Fix a broken button', 'Add a contact form', 'Make my site faster'];

export function greeting(hour: number): string {
  if (hour < 12) return 'Good morning';
  return hour < 18 ? 'Good afternoon' : 'Good evening';
}
