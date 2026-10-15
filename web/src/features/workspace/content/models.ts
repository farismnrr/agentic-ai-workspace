import type { ProviderKind } from '../logic/providers-api';

export interface KindChoice {
  value: ProviderKind;
  label: string;
  name: string;
  baseUrlHint: string;
  keyHelp: string;
}

export const kindChoices: KindChoice[] = [
  { value: 'anthropic', label: 'Anthropic (Claude)', name: 'Anthropic', baseUrlHint: 'https://api.anthropic.com', keyHelp: 'https://console.anthropic.com/settings/keys' },
  { value: 'openai', label: 'OpenAI', name: 'OpenAI', baseUrlHint: 'https://api.openai.com/v1', keyHelp: 'https://platform.openai.com/api-keys' },
  { value: 'gemini', label: 'Google (Gemini)', name: 'Gemini', baseUrlHint: 'https://generativelanguage.googleapis.com/v1beta/openai', keyHelp: 'https://aistudio.google.com/apikey' },
  { value: 'compatible', label: 'Gateway on this computer (for example 9router)', name: '9router', baseUrlHint: 'http://localhost:20128/v1', keyHelp: '' },
];

export const kindLabel = (kind: ProviderKind): string => kindChoices.find((choice) => choice.value === kind)?.label ?? kind;

export const modelsCopy = {
  defaultIntro: 'Used for every new task unless you pick another when you start it.',
  effortIntro: 'How hard the AI thinks before it answers. Higher is slower and costs more; a model that does not take a level will say so.',
  providersIntro: 'Your own keys. Tasks are billed by the provider, not by us.',
  none: { title: 'Add your first AI model', body: 'Add a provider with your own key so your AI team has a model to work with. Use Add above.' },
  noModels: { title: 'No models to choose from yet', body: 'Add a provider and verify its key, then pick a model here.' },
  contextIntro: 'How much a model can keep in mind at once. When a task gets close to it, the AI summarizes its notes and keeps going.',
  contextLabel: (model: string) => `Context window for ${model}`,
  failed: 'We could not load your providers. Please try again in a moment.',
  saved: 'Saved, but the provider did not accept this key. Check it and press Verify.',
  errors: {
    invalid_name: 'Give the provider a short name.',
    invalid_base_url: 'The address must be on this computer, like http://localhost:20128/v1.',
    address_fixed: 'This provider has a fixed address. Leave the address empty.',
    invalid_key: 'Enter the API key.',
    invalid_kind: 'Choose a provider.',
    too_many: 'You have too many providers. Remove one first.',
    invalid_context: 'Choose a size between 8,000 and 2,000,000 tokens.',
    verify_failed: 'The provider did not accept this key.',
  } as Record<string, string>,
  fallbackError: 'Something went wrong. Please try again in a moment.',
};

export const personaCopy = {
  talkStyles: ['Friendly', 'Formal', 'Short & direct'],
  languages: ['Bahasa Indonesia', 'English'],
  builtInRules: ['Ask before deleting anything', 'Never share passwords or keys', 'Show proof before marking a task done'],
  saved: 'Saved.',
  failed: 'We could not save that. Please try again.',
};

export const notificationsCopy = {
  description: 'Alerts only appear while this tab is in the background, and only on this device.',
  ask: 'Your browser will ask for permission.',
  blocked: 'Notifications are blocked for this site. Allow them in your browser settings, then reload.',
  unsupported: 'This browser does not support notifications.',
  asks: 'A task needs your answer or is ready for your review.',
  problems: 'A task failed or had to stop.',
};
