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
  { value: 'compatible', label: 'OpenAI-compatible (for example 9router)', name: '9router', baseUrlHint: 'http://localhost:20128/v1', keyHelp: '' },
];

export const kindLabel = (kind: ProviderKind): string => kindChoices.find((choice) => choice.value === kind)?.label ?? kind;

export const modelsCopy = {
  defaultIntro: 'Used for every new task unless you pick another when you start it.',
  providersIntro: 'Your own keys. Tasks are billed by the provider, not by us.',
  none: 'No provider yet. Add one so your AI team has a model to work with.',
  noModels: 'Add a provider and verify its key to choose a model.',
  failed: 'We could not load your providers. Please try again in a moment.',
  saved: 'Saved, but the provider did not accept this key. Check it and press Verify.',
  errors: {
    invalid_name: 'Give the provider a short name.',
    invalid_base_url: 'Enter the address like http://localhost:20128/v1.',
    invalid_key: 'Enter the API key.',
    invalid_kind: 'Choose a provider.',
    too_many: 'You have too many providers. Remove one first.',
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
