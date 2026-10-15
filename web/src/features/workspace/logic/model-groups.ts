import type { ModelOption } from './providers-api';

export interface ModelGroup {
  /** Shown above the models: the provider, and who serves them when a gateway says so. */
  label: string;
  models: ModelOption[];
}

/**
 * Models in groups, in the order the server listed them. A gateway such as 9router serves many providers' models
 * and reports who serves each (cc, cx, kr, ...), so they are split by that; nothing here knows the names, so a
 * provider added behind the gateway shows up as its own group.
 */
export function groupModels(options: ModelOption[]): ModelGroup[] {
  const groups = new Map<string, ModelGroup>();
  for (const option of options) {
    const label = option.group === '' ? option.providerName : `${option.providerName} · ${option.group}`;
    const group = groups.get(label) ?? { label, models: [] };
    group.models.push(option);
    groups.set(label, group);
  }
  return [...groups.values()];
}

/** The value of a model's option: which provider's copy of the model it is. */
export const modelValue = (option: ModelOption): string => `${option.providerId}::${option.model}`;
