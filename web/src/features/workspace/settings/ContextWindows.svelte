<script lang="ts">
  import { modelsCopy } from '../content/models';
  import { contextChoices, formatTokens } from '../logic/context-window';
  import { describeProviderError } from '../logic/provider-errors';
  import { setContextWindow, type Provider } from '../logic/providers-api';
  import SettingRow from './parts/SettingRow.svelte';
  import SettingsSection from './parts/SettingsSection.svelte';

  interface Props {
    providers: Provider[];
    onchange: () => void;
    onerror: (message: string) => void;
  }

  const { providers, onchange, onerror }: Props = $props();
  const usable = $derived(providers.filter((provider) => provider.enabled && provider.verified && provider.models.length > 0));

  async function pick(provider: Provider, model: string, value: string): Promise<void> {
    try {
      await setContextWindow(provider.id, model, Number(value));
    } catch (error) {
      onerror(describeProviderError(error));
    } finally {
      onchange();
    }
  }
</script>

{#if usable.length > 0}
  <SettingsSection title="Context window" description={modelsCopy.contextIntro}>
    {#each usable as provider (provider.id)}
      {#each provider.models as model (model)}
        {@const current = provider.contextWindows[model] ?? 200_000}
        <SettingRow label={model} description={provider.name} for="context-{provider.id}-{model}">
          <select
            id="context-{provider.id}-{model}"
            class="select select-sm w-32"
            aria-label={modelsCopy.contextLabel(model)}
            value={String(current)}
            onchange={(event) => pick(provider, model, event.currentTarget.value)}
          >
            {#each contextChoices(current) as size (size)}<option value={String(size)}>{formatTokens(size)} tokens</option>{/each}
          </select>
        </SettingRow>
      {/each}
    {/each}
  </SettingsSection>
{/if}
