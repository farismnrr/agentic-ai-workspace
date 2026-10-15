import type { ProviderKind } from './providers-api';

/** How hard the AI thinks. Empty means the provider's own default. */
export type Effort = '' | 'none' | 'minimal' | 'low' | 'medium' | 'high' | 'xhigh' | 'max';

type Level = Exclude<Effort, ''>;

const labels: Record<Level, string> = { none: 'None', minimal: 'Minimal', low: 'Low', medium: 'Medium', high: 'High', xhigh: 'Extra high', max: 'Max' };

// What each kind of provider is documented to accept. A model that does not take a level says so, and the
// message reaches the person as the AI model's own reply.
const levels: Record<ProviderKind, Level[]> = {
  anthropic: ['low', 'medium', 'high', 'xhigh', 'max'],
  openai: ['none', 'minimal', 'low', 'medium', 'high', 'xhigh', 'max'],
  compatible: ['none', 'minimal', 'low', 'medium', 'high', 'xhigh', 'max'],
  gemini: ['none', 'minimal', 'low', 'medium', 'high'],
};

export interface EffortChoice {
  value: Effort;
  label: string;
}

/** Default first, then the levels this kind of provider accepts. */
export function effortChoices(kind: ProviderKind | undefined): EffortChoice[] {
  const offered = levels[kind ?? 'compatible'];
  return [{ value: '', label: 'Default' }, ...offered.map((value) => ({ value, label: labels[value] }))];
}

const isLevel = (value: string): value is Level => value in labels;

/** The level as a word, or empty for the provider's default. */
export const effortLabel = (effort: string): string => (isLevel(effort) ? labels[effort] : effort);
