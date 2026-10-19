<script lang="ts">
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import { programs, subscriptionsCopy } from '../content/subscriptions';
  import { signOut, type Program } from '../logic/subscriptions-api';
  import IntegrationRow from './parts/IntegrationRow.svelte';
  import SubscriptionDialog from './SubscriptionDialog.svelte';

  interface Props {
    program: Program;
    connected: boolean;
    loading: boolean;
    /** The sign-in state could not be loaded. */
    problem: boolean;
    /** Called after signing in or out, so the lists that use the plan can refresh. */
    onchange: () => void;
  }

  const { program, connected, loading, problem, onchange }: Props = $props();
  const copy = $derived(programs[program]);
  let signingIn = $state(false);
  let confirming = $state(false);

  async function leave(): Promise<void> {
    confirming = false;
    try {
      await signOut(program);
    } finally {
      onchange();
    }
  }

  function done(): void {
    signingIn = false;
    onchange();
  }
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
    <button class="btn btn-outline btn-error btn-sm" type="button" onclick={() => (confirming = true)}>{subscriptionsCopy.signOut}</button>
  {:else if !problem}
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
