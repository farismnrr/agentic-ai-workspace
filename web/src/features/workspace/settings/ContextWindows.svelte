<script lang="ts">
  import { modelsCopy } from '../content/models';
  import { contextChoices, formatTokens } from '../logic/context-window';
  import { groupModels, modelValue } from '../logic/model-groups';
  import { describeProviderError } from '../logic/provider-errors';
  import { setContextWindow, type ModelOption, type Provider } from '../logic/providers-api';
  import SettingRow from './parts/SettingRow.svelte';
  import SettingsSection from './parts/SettingsSection.svelte';

  interface Props {
    /** The models a person can pick, in the same groups as the pickers. */
    options: ModelOption[];
    providers: Provider[];
    onchange: () => void;
    onerror: (message: string) => void;
  }

  const { options, providers, onchange, onerror }: Props = $props();

  const sizeOf = (option: ModelOption): number =>
    providers.find((provider) => provider.id === option.providerId)?.contextWindows[option.model] ?? 200_000;

  async function pick(option: ModelOption, value: string): Promise<void> {
    try {
      await setContextWindow(option.providerId, option.model, Number(value));
    } catch (error) {
      onerror(describeProviderError(error));
    } finally {
      onchange();
    }
  }
</script>

{#if options.length > 0}
  <SettingsSection title="Context window" description={modelsCopy.contextIntro}>
    {#each groupModels(options) as group (group.label)}
      <h3 class="pb-1 pt-4 text-xs font-semibold text-[var(--brand-text-secondary)]">{group.label}</h3>
      {#each group.models as option (modelValue(option))}
        {@const current = sizeOf(option)}
        <SettingRow label={option.model} for="context-{modelValue(option)}">
          <select
            id="context-{modelValue(option)}"
            class="select select-sm w-32"
            aria-label={modelsCopy.contextLabel(option.model)}
            value={String(current)}
            onchange={(event) => pick(option, event.currentTarget.value)}
          >
            {#each contextChoices(current) as size (size)}<option value={String(size)}>{formatTokens(size)} tokens</option>{/each}
          </select>
        </SettingRow>
      {/each}
    {/each}
  </SettingsSection>
{/if}
