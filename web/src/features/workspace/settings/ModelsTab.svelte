<script lang="ts">
  import EmptyState from '../../../shared/ui/EmptyState.svelte';
  import { onMount } from 'svelte';
  import Icon from '../../../shared/ui/Icon.svelte';
  import { modelsCopy, tierCopy } from '../content/models';
  import { programs, subscriptionsCopy } from '../content/subscriptions';
  import type { Program } from '../logic/subscriptions-api';
  import { loadPreferences, savePreferences, type ModelTier, type Preferences } from '../logic/preferences-api';
  import { loadModels, loadProviders, removeProvider, type ModelOption, type Provider } from '../logic/providers-api';
  import { describeProviderError } from '../logic/provider-errors';
  import AddProviderModal from './AddProviderModal.svelte';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import SettingsSection from './parts/SettingsSection.svelte';
  import ContextWindows from './ContextWindows.svelte';
  import ProviderRow from './ProviderRow.svelte';
  import SubscriptionDialog from './SubscriptionDialog.svelte';
  import TierRow from './TierRow.svelte';

  let providers: Provider[] = $state([]);
  let options: ModelOption[] = $state([]);
  let prefs: Preferences | undefined = $state();
  let loading = $state(true);
  let adding = $state(false);
  let pending: Provider | undefined = $state();
  let notice = $state('');
  let failed = $state(false);
  let signingIn: Program | undefined = $state();

  const pendingPlan = $derived<Program | undefined>(pending?.kind === 'plan' ? (pending.id.slice('plan:'.length) as Program) : undefined);
  const defaultTier = $derived<ModelTier>({ providerId: prefs?.defaultProviderId ?? '', model: prefs?.defaultModel ?? '', effort: prefs?.defaultEffort ?? '' });

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

  async function save(change: Partial<Preferences>): Promise<void> {
    if (prefs === undefined) return;
    try {
      prefs = await savePreferences({ ...prefs, ...change });
    } catch (error) {
      notice = describeProviderError(error);
    }
  }

  const saveDefault = (next: ModelTier): Promise<void> => save({ defaultProviderId: next.providerId, defaultModel: next.model, defaultEffort: next.effort });

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

  function startSignIn(program: Program): void {
    signingIn = program;
  }

  function signedIn(): void {
    signingIn = undefined;
    void refresh();
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
  <SettingsSection title="Models" description={tierCopy.intro}>
    {#if options.length === 0}
      <EmptyState compact icon="key" title={modelsCopy.noModels.title} body={modelsCopy.noModels.body} />
    {:else if prefs !== undefined}
      <TierRow id="tier-thinking" label={tierCopy.tiers.thinking.label} note={tierCopy.tiers.thinking.note} value={prefs.thinking} {options} {providers} emptyLabel={tierCopy.unsetThinking} onchange={(next: ModelTier) => save({ thinking: next })} />
      <TierRow id="tier-default" label={tierCopy.tiers.default.label} note={tierCopy.tiers.default.note} value={defaultTier} {options} {providers} emptyLabel="Choose when I start a task" onchange={saveDefault} />
      <TierRow id="tier-fast" label={tierCopy.tiers.fast.label} note={tierCopy.tiers.fast.note} value={prefs.fast} {options} {providers} emptyLabel={tierCopy.unsetFast} onchange={(next: ModelTier) => save({ fast: next })} />
    {/if}
  </SettingsSection>

  <ContextWindows {options} {providers} onchange={refresh} onerror={showNotice} />

  <SettingsSection title="Connected providers" description={modelsCopy.providersIntro}>
    {#snippet action()}
      <button class="btn btn-ghost btn-sm" type="button" aria-label="Add provider" onclick={() => (adding = true)}><Icon name="plus" class="size-4" /> Add</button>
    {/snippet}
    {#if notice !== ''}<p class="py-3 text-sm text-error" role="alert">{notice}</p>{/if}
    {#if providers.length === 0}
      <EmptyState compact icon="key" title={modelsCopy.none.title} body={modelsCopy.none.body} />
    {:else}
      <ul>
        {#each providers as provider (provider.id)}
          <ProviderRow {provider} onchange={refresh} onremove={() => (pending = provider)} onerror={showNotice} />
        {/each}
      </ul>
    {/if}
  </SettingsSection>

{/if}

<AddProviderModal bind:show={adding} onadded={added} onplan={startSignIn} />
{#if signingIn !== undefined}
  <SubscriptionDialog program={signingIn} open onclose={() => (signingIn = undefined)} ondone={signedIn} />
{/if}
<ConfirmModal
  open={pending !== undefined}
  title={pendingPlan === undefined ? `Remove ${pending?.name ?? ''}?` : subscriptionsCopy.signOutDialog.title(programs[pendingPlan].name)}
  body={pendingPlan === undefined ? 'Its key is deleted. Tasks that use its models will need another provider.' : subscriptionsCopy.signOutDialog.body}
  confirmLabel={pendingPlan === undefined ? 'Remove provider' : subscriptionsCopy.signOutDialog.confirm}
  cancelLabel={pendingPlan === undefined ? 'Keep it' : subscriptionsCopy.signOutDialog.cancel}
  oncancel={() => (pending = undefined)}
  onconfirm={confirmRemove}
/>
