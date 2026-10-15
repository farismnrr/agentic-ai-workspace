<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
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
      <!-- Closed until opened: a gateway can serve dozens of models, and most people never change these. -->
      <details class="group">
        <summary class="flex cursor-pointer list-none items-center gap-2 py-3">
          <span class="flex-1 font-medium">{group.label}</span>
          <span class="text-xs text-[var(--brand-text-tertiary)]">{group.models.length} {group.models.length === 1 ? 'model' : 'models'}</span>
          <Icon name="chevron-down" class="size-4 text-[var(--brand-text-tertiary)] transition-transform motion-reduce:transition-none group-open:rotate-180" />
        </summary>
        <div class="divide-y divide-[var(--brand-border)] border-t border-[var(--brand-border)]">
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
        </div>
      </details>
    {/each}
  </SettingsSection>
{/if}
