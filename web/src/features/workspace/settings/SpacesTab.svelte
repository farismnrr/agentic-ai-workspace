<script lang="ts">
  import EmptyState from '../../../shared/ui/EmptyState.svelte';
  import { onMount } from 'svelte';
  import { spacesCopy } from '../content/connected';
  import { loadSpaces, type ConnectorSpace } from '../logic/connectors-api';
  import ConnectorAddress from './ConnectorAddress.svelte';
  import SkipApprovals from './SkipApprovals.svelte';
  import SpaceTools from './SpaceTools.svelte';

  let spaces: ConnectorSpace[] = $state([]);
  let selected = $state('');
  let loading = $state(true);
  let failed = $state(false);
  const space = $derived(spaces.find((item) => item.slug === selected) ?? spaces[0]);

  onMount(async () => {
    try {
      spaces = await loadSpaces();
    } catch {
      failed = true;
    } finally {
      loading = false;
    }
  });
</script>

<p class="mb-4 text-xs text-[var(--brand-text-tertiary)]">{spacesCopy.intro}</p>
{#if loading}
  <span class="loading loading-spinner loading-sm" aria-label="Loading"></span>
{:else if failed}
  <p class="text-error" role="alert">{spacesCopy.failed}</p>
{:else if space === undefined}
  <EmptyState bordered icon="box" title={spacesCopy.empty.title} body={spacesCopy.empty.body} />
{:else}
  <div class="mb-8 grid grid-cols-1 items-start gap-4 md:grid-cols-[11rem_minmax(0,1fr)]">
    <div>
      <h2 class="mb-1 px-1 text-sm font-semibold">Spaces</h2>
      <ul class="flex gap-1 overflow-x-auto md:flex-col">
        {#each spaces as item (item.slug)}
          <li>
            <button class="flex h-9 w-full shrink-0 items-center rounded-lg px-2 text-left hover:bg-base-100/70 {space.slug === item.slug ? 'bg-base-300/60 font-medium' : ''}" type="button" aria-pressed={space.slug === item.slug} onclick={() => (selected = item.slug)}>
              <span class="flex-1 truncate">{item.name}</span>
            </button>
          </li>
        {/each}
      </ul>
    </div>
    <section class="min-w-0 rounded-xl border border-[var(--brand-border)] bg-base-100" aria-label="{space.name} tools">
      <h2 class="border-b border-[var(--brand-border)] px-4 py-3 text-sm font-semibold">{space.name}</h2>
      <ConnectorAddress url={space.url} />
      {#key space.slug}<SkipApprovals {space} />{/key}
      <SpaceTools {space} />
    </section>
  </div>
{/if}
