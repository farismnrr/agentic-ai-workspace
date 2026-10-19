<script lang="ts">
  import EmptyState from '../../../shared/ui/EmptyState.svelte';
  import { onMount } from 'svelte';
  import Icon from '../../../shared/ui/Icon.svelte';
  import { modelsCopy } from '../content/models';
  import { planGroupLabel, programs, subscriptionsCopy } from '../content/subscriptions';
  import { loadConnectedPlans, signOut, type Program } from '../logic/subscriptions-api';
  import { effortChoices, hasEffortChoice } from '../logic/effort';
  import { groupModels, modelValue } from '../logic/model-groups';
  import { loadPreferences, savePreferences, type Preferences } from '../logic/preferences-api';
  import { loadModels, loadProviders, removeProvider, type ModelOption, type Provider } from '../logic/providers-api';
  import { describeProviderError } from '../logic/provider-errors';
  import AddProviderModal from './AddProviderModal.svelte';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import SettingRow from './parts/SettingRow.svelte';
  import SettingsSection from './parts/SettingsSection.svelte';
  import ContextWindows from './ContextWindows.svelte';
  import PlanRow from './PlanRow.svelte';
  import ProviderRow from './ProviderRow.svelte';
  import SubscriptionDialog from './SubscriptionDialog.svelte';

  let providers: Provider[] = $state([]);
  let options: ModelOption[] = $state([]);
  let prefs: Preferences | undefined = $state();
  let loading = $state(true);
  let adding = $state(false);
  let pending: Provider | undefined = $state();
  let notice = $state('');
  let failed = $state(false);
  let plans: Program[] = $state([]);
  let signingIn: Program | undefined = $state();
  let leaving: Program | undefined = $state();

  const chosen = $derived(prefs === undefined || prefs.defaultModel === '' ? '' : `${prefs.defaultProviderId}::${prefs.defaultModel}`);

  async function refresh(): Promise<void> {
    try {
      [providers, options, prefs, plans] = await Promise.all([loadProviders(), loadModels(), loadPreferences(), loadConnectedPlans()]);
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

  const defaultLevels = $derived(providers.find((provider) => provider.id === prefs?.defaultProviderId)?.efforts[prefs?.defaultModel ?? '']);

  async function pickEffort(value: string): Promise<void> {
    if (prefs === undefined) return;
    try {
      prefs = await savePreferences({ ...prefs, defaultEffort: value });
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

  async function confirmLeave(): Promise<void> {
    const target = leaving;
    leaving = undefined;
    if (target === undefined) return;
    try {
      await signOut(target);
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
  <SettingsSection title="Default model" description={modelsCopy.defaultIntro}>
    {#if options.length === 0 && plans.length === 0}
      <EmptyState compact icon="key" title={modelsCopy.noModels.title} body={modelsCopy.noModels.body} />
    {:else}
      <SettingRow label="Model" for="default-model">
        <select id="default-model" class="select select-sm w-72" value={chosen} onchange={(event) => pick(event.currentTarget.value)}>
          <option value="">Choose when I start a task</option>
          {#if plans.length > 0}
            <optgroup label={planGroupLabel}>
              {#each plans as program (program)}<option value="plan:{program}::default">{programs[program].name}</option>{/each}
            </optgroup>
          {/if}
          {#each groupModels(options) as group (group.label)}
            <optgroup label={group.label}>
              {#each group.models as option (modelValue(option))}<option value={modelValue(option)}>{option.model}</option>{/each}
            </optgroup>
          {/each}
        </select>
      </SettingRow>
      {#if hasEffortChoice(defaultLevels)}
        <SettingRow label="Effort" description={modelsCopy.effortIntro} for="default-effort">
          <select id="default-effort" class="select select-sm w-40" value={prefs?.defaultEffort ?? ''} onchange={(event) => pickEffort(event.currentTarget.value)}>
            {#each effortChoices(defaultLevels) as choice (choice.value)}<option value={choice.value}>{choice.label}</option>{/each}
          </select>
        </SettingRow>
      {/if}
    {/if}
  </SettingsSection>

  <ContextWindows {options} {providers} onchange={refresh} onerror={showNotice} />

  <SettingsSection title="Connected providers" description={modelsCopy.providersIntro}>
    {#snippet action()}
      <button class="btn btn-ghost btn-sm" type="button" aria-label="Add provider" onclick={() => (adding = true)}><Icon name="plus" class="size-4" /> Add</button>
    {/snippet}
    {#if notice !== ''}<p class="py-3 text-sm text-error" role="alert">{notice}</p>{/if}
    {#if providers.length === 0 && plans.length === 0}
      <EmptyState compact icon="key" title={modelsCopy.none.title} body={modelsCopy.none.body} />
    {:else}
      <ul>
        {#each plans as program (program)}
          <PlanRow {program} onremove={() => (leaving = program)} />
        {/each}
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
  open={leaving !== undefined}
  title={subscriptionsCopy.signOutDialog.title(leaving === undefined ? '' : programs[leaving].name)}
  body={subscriptionsCopy.signOutDialog.body}
  confirmLabel={subscriptionsCopy.signOutDialog.confirm}
  cancelLabel={subscriptionsCopy.signOutDialog.cancel}
  oncancel={() => (leaving = undefined)}
  onconfirm={confirmLeave}
/>
<ConfirmModal
  open={pending !== undefined}
  title="Remove {pending?.name ?? ''}?"
  body="Its key is deleted. Tasks that use its models will need another provider."
  confirmLabel="Remove provider"
  cancelLabel="Keep it"
  oncancel={() => (pending = undefined)}
  onconfirm={confirmRemove}
/>
