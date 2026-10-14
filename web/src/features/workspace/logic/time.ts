const minute = 60_000;
const hour = 60 * minute;
const day = 24 * hour;

export type DayGroup = 'Today' | 'Yesterday' | 'Earlier';

/** A short, friendly distance from now: "just now", "5 min ago", "3 h ago", "Yesterday". */
export function timeAgo(iso: string, now = Date.now()): string {
  const elapsed = now - new Date(iso).getTime();
  if (elapsed < minute) return 'just now';
  if (elapsed < hour) return `${String(Math.floor(elapsed / minute))} min ago`;
  if (elapsed < day) return `${String(Math.floor(elapsed / hour))} h ago`;
  if (elapsed < 2 * day) return 'Yesterday';
  return new Date(iso).toLocaleDateString(undefined, { dateStyle: 'medium' });
}

/** The time of day, such as 09:14. */
export const clock = (iso: string): string => new Date(iso).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });

export function dayGroup(iso: string, now = Date.now()): DayGroup {
  const startOfToday = new Date(now).setHours(0, 0, 0, 0);
  const moment = new Date(iso).getTime();
  if (moment >= startOfToday) return 'Today';
  return moment >= startOfToday - day ? 'Yesterday' : 'Earlier';
}
