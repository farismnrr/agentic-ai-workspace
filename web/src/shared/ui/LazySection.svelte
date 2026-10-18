<script lang="ts">
  import type { Component } from 'svelte';

  interface Props {
    load: () => Promise<{ default: Component }>;
    minHeight?: string;
    id?: string;
  }

  const { load, minHeight = '24rem', id }: Props = $props();

  const maxTries = 4;

  let host = $state<HTMLElement>();
  let Loaded = $state<Component>();
  let failed = $state(false);

  /** Loads the section, trying again a few times: a chunk request can fail on a slow or dropped connection. */
  async function fetchSection(attempt = 1): Promise<void> {
    try {
      Loaded = (await load()).default;
      failed = false;
    } catch {
      if (attempt >= maxTries) {
        failed = true;
        return;
      }
      await new Promise((resolve) => setTimeout(resolve, attempt * 800));
      await fetchSection(attempt + 1);
    }
  }

  function retry(): void {
    failed = false;
    void fetchSection();
  }

  $effect(() => {
    if (host === undefined) return;
    if (!('IntersectionObserver' in window)) {
      void fetchSection();
      return;
    }
    const io = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          io.disconnect();
          void fetchSection();
        }
      },
      { rootMargin: '400px 0px' },
    );
    io.observe(host);
    return () => {
      io.disconnect();
    };
  });
</script>

{#if Loaded}
  <div {id}><Loaded /></div>
{:else}
  <div bind:this={host} {id} style:min-height={minHeight} class="flex items-center justify-center">
    {#if failed}
      <button type="button" class="btn btn-outline btn-sm" onclick={retry}>This part did not load. Try again</button>
    {/if}
  </div>
{/if}
