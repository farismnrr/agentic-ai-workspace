<script lang="ts">
  import { onMount } from 'svelte';
  import { effortChoices, hasEffortChoice } from '../logic/effort';
  import { groupModels, modelValue } from '../logic/model-groups';
  import { loadPreferences, type Preferences } from '../logic/preferences-api';
  import { loadModels, loadProviders, type ModelOption, type Provider } from '../logic/providers-api';

  interface Props {
    /** "providerId::model", or empty for the default model. */
    model?: string | undefined;
    effort?: string | undefined;
    disabled?: boolean | undefined;
    /** False for a role that cannot run on a plan. */
    plans?: boolean | undefined;
  }

  let { model = $bindable(''), effort = $bindable(''), disabled = false, plans = true }: Props = $props();
  let all: ModelOption[] = $state([]);
  const options = $derived(plans ? all : all.filter((option) => !option.providerId.startsWith('plan:')));
  let providers: Provider[] = $state([]);
  let prefs: Preferences | undefined = $state();

  // The model the task will run on: the one picked, else the person's default, else the first one offered.
  const target = $derived.by(() => {
    if (model !== '') {
      const [id = '', name = ''] = model.split('::');
      return { id, name };
    }
    if (prefs !== undefined && prefs.defaultModel !== '') return { id: prefs.defaultProviderId, name: prefs.defaultModel };
    const first = options[0];
    return first === undefined ? undefined : { id: first.providerId, name: first.model };
  });
  const levels = $derived(target === undefined ? undefined : providers.find((item) => item.id === target.id)?.efforts[target.name]);
  const choices = $derived(effortChoices(levels));

  // A level the newly chosen model's provider does not take falls back to the default.
  $effect(() => {
    if (effort !== '' && !choices.some((choice) => choice.value === effort)) effort = '';
  });

  onMount(async () => {
    try {
      [all, providers, prefs] = await Promise.all([loadModels(), loadProviders(), loadPreferences()]);
    } catch {
      all = [];
    }
  });
</script>

{#if options.length > 0}
  <label class="sr-only" for="composer-model">Model</label>
  <select id="composer-model" class="select select-ghost select-sm w-auto max-w-44 rounded-full {model === '' ? 'text-[var(--brand-text-secondary)]' : 'font-medium'}" bind:value={model} {disabled}>
    <option value="">Default model</option>
    {#each groupModels(options) as group (group.label)}
      <optgroup label={group.label}>
        {#each group.models as option (modelValue(option))}<option value={modelValue(option)}>{option.model}</option>{/each}
      </optgroup>
    {/each}
  </select>
  {#if hasEffortChoice(levels)}
    <label class="sr-only" for="composer-effort">Effort</label>
    <select id="composer-effort" class="select select-ghost select-sm w-auto rounded-full {effort === '' ? 'text-[var(--brand-text-secondary)]' : 'font-medium'}" bind:value={effort} {disabled}>
      {#each choices as choice (choice.value)}<option value={choice.value}>{choice.value === '' ? 'Default effort' : `${choice.label} effort`}</option>{/each}
    </select>
  {/if}
{/if}
