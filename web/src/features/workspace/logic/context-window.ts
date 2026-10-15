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
