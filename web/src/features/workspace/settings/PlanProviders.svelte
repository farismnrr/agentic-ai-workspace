<script lang="ts">
  import { onMount } from 'svelte';
  import { subscriptionsCopy } from '../content/subscriptions';
  import { loadSubscription, type Program } from '../logic/subscriptions-api';
  import SettingsSection from './parts/SettingsSection.svelte';
  import SubscriptionRow from './SubscriptionRow.svelte';

  interface Props {
    /** The programs the person is signed in to; the default model can be one of them. */
    connected?: Program[] | undefined;
    onchange: () => void;
  }

  let { connected = $bindable([]), onchange }: Props = $props();
  const all: Program[] = ['claude', 'chatgpt'];
  let loading = $state(true);
  let problem = $state(false);

  async function refresh(): Promise<void> {
    try {
      const states = await Promise.all(all.map((program) => loadSubscription(program)));
      connected = all.filter((_, index) => states[index]?.connected === true);
      problem = false;
    } catch {
      problem = true;
    } finally {
      loading = false;
    }
  }

  function changed(): void {
    void refresh().then(onchange);
  }

  onMount(() => {
    void refresh();
  });
</script>

<SettingsSection title={subscriptionsCopy.title} description={subscriptionsCopy.intro}>
  <ul class="list mt-3 rounded-box border border-base-300 bg-base-100">
    {#each all as program (program)}
      <SubscriptionRow {program} connected={connected.includes(program)} {loading} {problem} onchange={changed} />
    {/each}
  </ul>
</SettingsSection>
