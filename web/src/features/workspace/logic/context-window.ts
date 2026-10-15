const CHOICES = [32_000, 128_000, 200_000, 272_000, 400_000, 1_000_000];

/** "200K" or "1M". */
export function formatTokens(tokens: number): string {
  if (tokens >= 1_000_000) return `${String(tokens / 1_000_000)}M`;
  return `${String(Math.round(tokens / 1000))}K`;
}

/** The sizes to offer, always including the current one so it can be shown as selected. */
export function contextChoices(current: number): number[] {
  return CHOICES.includes(current) ? CHOICES : [...CHOICES, current].sort((a, b) => a - b);
}

/** Share of the window in use, 0 to 100. */
export const contextPercent = (used: number, window: number): number => (window > 0 ? Math.min(100, Math.round((used / window) * 100)) : 0);

/** Warns once the window is nearly full, which is when the AI is about to summarize. */
export const contextTone = (percent: number): 'progress-primary' | 'progress-warning' => (percent >= 75 ? 'progress-warning' : 'progress-primary');
