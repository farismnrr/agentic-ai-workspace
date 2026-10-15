/** How hard the AI thinks, as a level the provider understands. Empty means the provider's own default. */
export type Effort = string;

const labels: Record<string, string> = { none: 'None', minimal: 'Minimal', low: 'Low', medium: 'Medium', high: 'High', xhigh: 'Extra high', max: 'Max' };

export interface EffortChoice {
  value: Effort;
  label: string;
}

/** The level as a word, or empty for the provider's default. A level we have no word for is shown as it is. */
export const effortLabel = (effort: string): string => labels[effort] ?? effort;

/**
 * Default first, then the levels the server says suit the model. The server works them out from what the provider
 * reported about the model, so a new model or a new provider behind a gateway needs no change here.
 */
export function effortChoices(levels: string[] | undefined): EffortChoice[] {
  return [{ value: '', label: 'Default' }, ...(levels ?? []).map((value) => ({ value, label: effortLabel(value) }))];
}

/** Whether to offer a choice at all: not for a model the provider says does not reason. */
export const hasEffortChoice = (levels: string[] | undefined): boolean => levels === undefined || levels.length > 0;
