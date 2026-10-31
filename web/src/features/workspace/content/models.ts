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
  { value: 'compatible', label: 'Local OpenAI-compatible API (own API key only)', name: 'Local API', baseUrlHint: 'http://localhost:9000/v1', keyHelp: '' },
];

export const kindLabel = (kind: ProviderKind): string => kindChoices.find((choice) => choice.value === kind)?.label ?? kind;

/**
 * Legacy aliases reported by local OpenAI-compatible model gateways. A gateway may return a short provider alias;
 * known aliases are labelled while others keep the original value. Models are always read from the provider.
 */
export const gatewayProviderNames: Record<string, string> = {
  cx: 'OpenAI Codex',
  cc: 'Claude Code',
  oc: 'OpenCode Free',
};

/** The three model tiers: set a model once, and every role on the tier follows it. */
export const tierCopy = {
  intro: 'Models are shared by all roles using the same tier.',
  unsetThinking: 'Use the default model',
  unsetFast: 'Use the default model',
  tiers: {
    thinking: { label: 'Thinking', note: 'For complex work; often slower and more costly.' },
    default: { label: 'Default', note: 'Used for new tasks and roles without a custom tier.' },
    fast: { label: 'Fast', note: 'For quicker, simpler tasks.' },
  },
};

export const modelsCopy = {
  refreshTip: 'Check the key and refresh models and context limits.',
  providersIntro: 'Provider billing applies to API key usage.',
  none: { title: 'Add your first AI model', body: 'Add a provider and API key to get started.' },
  noModels: { title: 'No models to choose from yet', body: 'Add a provider, verify its key, then choose a model.' },
  contextIntro: 'Maximum tokens a model can use before notes are summarized.',
  contextLabel: (model: string) => `Context window for ${model}`,
  failed: 'We could not load your providers. Please try again in a moment.',
  saved: 'Saved, but the provider did not accept this key. Check it and press Verify.',
  errors: {
    invalid_name: 'Give the provider a short name.',
    invalid_base_url: 'The address must be on this computer, like http://localhost:20128/v1.',
    address_fixed: 'This provider has a fixed address. Leave the address empty.',
    invalid_key: 'Enter the API key.',
    invalid_kind: 'Choose a provider.',
    account_relay_disabled: '9router is no longer supported. Use your own provider API key or an official coding CLI.',
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
  description: 'Only on this device, when this tab is in the background.',
  blocked: 'Enable notifications in your browser settings, then reload.',
  unsupported: 'This browser does not support notifications.',
};
