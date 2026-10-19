<script lang="ts">
  import { onMount } from 'svelte';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import { programs, subscriptionsCopy } from '../content/subscriptions';
  import { isPlanDefault, setPlanDefault } from '../logic/plan-default';
  import { loadSubscription, signOut, type Program } from '../logic/subscriptions-api';
  import IntegrationRow from './parts/IntegrationRow.svelte';
  import SubscriptionDialog from './SubscriptionDialog.svelte';

  interface Props {
    program: Program;
    /** Changes whenever another row changed the default, so every row shows the current one. */
    version: number;
    onchange: () => void;
  }

  const { program, version, onchange }: Props = $props();
  const copy = $derived(programs[program]);
  let connected = $state(false);
  let loading = $state(true);
  let problem = $state(false);
  let signingIn = $state(false);
  let confirming = $state(false);
  let isDefault = $state(false);

  async function refresh(): Promise<void> {
    try {
      connected = (await loadSubscription(program)).connected;
      isDefault = connected && (await isPlanDefault(program));
      problem = false;
    } catch {
      problem = true;
    } finally {
      loading = false;
    }
  }

  async function leave(): Promise<void> {
    confirming = false;
    try {
      await signOut(program);
    } finally {
      await refresh();
    }
  }

  async function toggle(on: boolean): Promise<void> {
    try {
      await setPlanDefault(program, on);
    } finally {
      onchange();
    }
  }

  function done(): void {
    signingIn = false;
    void refresh();
  }

  onMount(() => {
    void refresh();
  });

  $effect(() => {
    if (version > 0) void refresh();
  });
</script>

<IntegrationRow
  logo="plan"
  title={copy.name}
  description={problem ? subscriptionsCopy.loadFailed : connected ? copy.connected : copy.description}
  {problem}
>
  {#if loading}
    <span class="loading loading-spinner loading-sm" aria-label="Loading"></span>
  {:else if connected}
    <label class="flex items-center gap-2 text-sm">
      <input type="checkbox" class="toggle toggle-primary toggle-sm" checked={isDefault} onchange={(event) => void toggle(event.currentTarget.checked)} />
      {subscriptionsCopy.useForNew}
    </label>
    <button class="btn btn-outline btn-error btn-sm" type="button" onclick={() => (confirming = true)}>{subscriptionsCopy.signOut}</button>
  {:else}
    <button class="btn btn-primary btn-sm" type="button" onclick={() => (signingIn = true)}>{subscriptionsCopy.signIn}</button>
  {/if}
</IntegrationRow>
<SubscriptionDialog {program} open={signingIn} onclose={() => (signingIn = false)} ondone={done} />
<ConfirmModal
  open={confirming}
  title={subscriptionsCopy.signOutDialog.title(copy.name)}
  body={subscriptionsCopy.signOutDialog.body}
  confirmLabel={subscriptionsCopy.signOutDialog.confirm}
  cancelLabel={subscriptionsCopy.signOutDialog.cancel}
  oncancel={() => (confirming = false)}
  onconfirm={leave}
/>
