<script lang="ts">
  import { effortChoices, hasEffortChoice } from '../logic/effort';
  import { groupModels, modelValue } from '../logic/model-groups';
  import type { ModelTier } from '../logic/preferences-api';
  import type { ModelOption, Provider } from '../logic/providers-api';
  import SettingRow from './parts/SettingRow.svelte';

  interface Props {
    id: string;
    label: string;
    note: string;
    value: ModelTier;
    options: ModelOption[];
    providers: Provider[];
    /** What an empty choice says, such as "Use the default model". */
    emptyLabel: string;
    onchange: (next: ModelTier) => void;
  }

  const { id, label, note, value, options, providers, emptyLabel, onchange }: Props = $props();

  const chosen = $derived(value.model === '' ? '' : `${value.providerId}::${value.model}`);
  const levels = $derived(providers.find((provider) => provider.id === value.providerId)?.efforts[value.model]);

  function pick(next: string): void {
    const [providerId = '', model = ''] = next.split('::');
    onchange({ providerId: next === '' ? '' : providerId, model: next === '' ? '' : model, effort: '' });
  }
</script>

<SettingRow {label} description={note} for="{id}-model">
  <div class="flex flex-wrap items-center gap-2">
    <select id="{id}-model" class="select select-sm w-64" value={chosen} onchange={(event) => { pick(event.currentTarget.value); }}>
      <option value="">{emptyLabel}</option>
      {#each groupModels(options) as group (group.label)}
        <optgroup label={group.label}>
          {#each group.models as option (modelValue(option))}<option value={modelValue(option)}>{option.model}</option>{/each}
        </optgroup>
      {/each}
    </select>
    {#if value.model !== '' && hasEffortChoice(levels)}
      <select class="select select-sm w-32" aria-label="{label} effort" value={value.effort} onchange={(event) => { onchange({ ...value, effort: event.currentTarget.value }); }}>
        {#each effortChoices(levels) as choice (choice.value)}<option value={choice.value}>{choice.label}</option>{/each}
      </select>
    {/if}
  </div>
</SettingRow>
