import { gatewayProviderNames } from '../content/models';
import type { ModelOption } from './providers-api';

export interface ModelGroup {
  /** Shown above the models: the provider, and who serves them when a gateway says so. */
  label: string;
  models: ModelOption[];
}

/**
 * Models in groups, in the order the server listed them. A local API can report a provider alias,
 * so models are grouped by that. The few best-known provider aliases get a proper
 * name; any other one shows up as its own group under the alias it reported.
 */
export function groupModels(options: ModelOption[]): ModelGroup[] {
  const groups = new Map<string, ModelGroup>();
  for (const option of options) {
    const label = option.group === '' ? option.providerName : `${option.providerName} · ${gatewayProviderNames[option.group] ?? option.group}`;
    const group = groups.get(label) ?? { label, models: [] };
    group.models.push(option);
    groups.set(label, group);
  }
  return [...groups.values()];
}

/** The value of a model's option: which provider's copy of the model it is. */
export const modelValue = (option: ModelOption): string => `${option.providerId}::${option.model}`;
