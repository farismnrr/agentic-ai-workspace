<script lang="ts">
  import type { Component } from 'svelte';

  interface Props {
    load: () => Promise<{ default: Component }>;
    minHeight?: string;
    id?: string;
  }

  const { load, minHeight = '24rem', id }: Props = $props();

  let host = $state<HTMLElement>();
  let Loaded = $state<Component>();

  $effect(() => {
    if (host === undefined) return;
    if (!('IntersectionObserver' in window)) {
      void load().then((module) => (Loaded = module.default));
      return;
    }
    const io = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          io.disconnect();
          void load().then((module) => (Loaded = module.default));
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
  <Loaded />
{:else}
  <div bind:this={host} {id} style:min-height={minHeight} aria-hidden="true"></div>
{/if}
