import { gatewayProviderNames } from '../content/models';
import { programs } from '../content/subscriptions';
import type { ModelOption } from './providers-api';
import type { ConnectedPlan, PlanModel, Program } from './subscriptions-api';

export interface ModelGroup {
  /** Shown above the models: the provider, and who serves them when a gateway says so. */
  label: string;
  models: ModelOption[];
}

/**
 * Models in groups, in the order the server listed them. A gateway such as 9router serves many providers' models
 * and reports who serves each (cc, cx, kr, ...), so they are split by that. The few best-known providers get a proper
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

/** The value of a plan's option, in the same "provider::model" shape: the plan stands in for the provider. */
export const planValue = (program: Program, model: PlanModel): string => `plan:${program}::${model.id}`;

/** The program's own choice is just the program; any other is the program and the model. */
export const planModelLabel = (program: Program, model: PlanModel): string =>
  model.id === 'default' ? programs[program].name : `${programs[program].name} · ${model.name}`;

/** The effort levels of a plan's model, or undefined when the provider is not a plan the person is signed in to. */
export function planLevels(plans: ConnectedPlan[], providerId: string, model: string): string[] | undefined {
  return plans.find((plan) => `plan:${plan.program}` === providerId)?.models.find((item) => item.id === model)?.efforts;
}
