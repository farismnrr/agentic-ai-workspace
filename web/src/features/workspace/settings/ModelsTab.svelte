<script lang="ts">
  import { onMount } from 'svelte';
  import Icon from '../../../shared/ui/Icon.svelte';
  import { modelsCopy } from '../content/models';
  import { loadPreferences, savePreferences, type Preferences } from '../logic/preferences-api';
  import { loadModels, loadProviders, removeProvider, type ModelOption, type Provider } from '../logic/providers-api';
  import { describeProviderError } from '../logic/provider-errors';
  import AddProviderModal from './AddProviderModal.svelte';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import SettingRow from './parts/SettingRow.svelte';
  import SettingsSection from './parts/SettingsSection.svelte';
  import ProviderRow from './ProviderRow.svelte';

  let providers: Provider[] = $state([]);
  let options: ModelOption[] = $state([]);
  let prefs: Preferences | undefined = $state();
  let loading = $state(true);
  let adding = $state(false);
  let pending: Provider | undefined = $state();
  let notice = $state('');
  let failed = $state(false);

  const encode = (option: ModelOption): string => `${option.providerId}::${option.model}`;
  const chosen = $derived(prefs === undefined || prefs.defaultModel === '' ? '' : `${prefs.defaultProviderId}::${prefs.defaultModel}`);

  async function refresh(): Promise<void> {
    try {
      [providers, options, prefs] = await Promise.all([loadProviders(), loadModels(), loadPreferences()]);
      failed = false;
    } catch {
      failed = true;
    } finally {
      loading = false;
    }
  }

  async function pick(value: string): Promise<void> {
    if (prefs === undefined) return;
    const [providerId = '', model = ''] = value.split('::');
    try {
      prefs = await savePreferences({ ...prefs, defaultProviderId: providerId, defaultModel: model });
    } catch (error) {
      notice = describeProviderError(error);
    }
  }

  async function confirmRemove(): Promise<void> {
    const target = pending;
    pending = undefined;
    if (target === undefined) return;
    try {
      await removeProvider(target.id);
    } catch (error) {
      notice = describeProviderError(error);
    }
    await refresh();
  }

  const showNotice = (message: string): void => {
    notice = message;
  };

  function added(verified: boolean): void {
    notice = verified ? '' : modelsCopy.saved;
    void refresh();
  }

  onMount(refresh);
</script>

{#if loading}
  <span class="loading loading-spinner loading-sm" aria-label="Loading"></span>
{:else if failed}
  <p class="text-error" role="alert">{modelsCopy.failed}</p>
{:else}
  <SettingsSection title="Default model" description={modelsCopy.defaultIntro}>
    {#if options.length === 0}
      <p class="py-3 text-[var(--brand-text-secondary)]">{modelsCopy.noModels}</p>
    {:else}
      <SettingRow label="Model" for="default-model">
        <select id="default-model" class="select select-sm w-72" value={chosen} onchange={(event) => pick(event.currentTarget.value)}>
          <option value="">Choose when I start a task</option>
          {#each options as option (encode(option))}<option value={encode(option)}>{option.model} · {option.providerName}</option>{/each}
        </select>
      </SettingRow>
    {/if}
  </SettingsSection>

  <SettingsSection title="Connected providers" description={modelsCopy.providersIntro}>
    {#snippet action()}
      <button class="btn btn-ghost btn-sm" type="button" aria-label="Add provider" onclick={() => (adding = true)}><Icon name="plus" class="size-4" /> Add</button>
    {/snippet}
    {#if notice !== ''}<p class="py-3 text-sm text-error" role="alert">{notice}</p>{/if}
    {#if providers.length === 0}
      <p class="py-3 text-[var(--brand-text-secondary)]">{modelsCopy.none}</p>
    {:else}
      <ul>
        {#each providers as provider (provider.id)}
          <ProviderRow {provider} onchange={refresh} onremove={() => (pending = provider)} onerror={showNotice} />
        {/each}
      </ul>
    {/if}
  </SettingsSection>
{/if}

<AddProviderModal bind:show={adding} onadded={added} />
<ConfirmModal
  open={pending !== undefined}
  title="Remove {pending?.name ?? ''}?"
  body="Its key is deleted. Tasks that use its models will need another provider."
  confirmLabel="Remove provider"
  cancelLabel="Keep it"
  oncancel={() => (pending = undefined)}
  onconfirm={confirmRemove}
/>
