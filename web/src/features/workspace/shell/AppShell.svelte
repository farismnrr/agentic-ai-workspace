<script lang="ts">
  import type { Snippet } from 'svelte';
  import CommandPalette from './CommandPalette.svelte';
  import { paletteOpen, type Crumb } from './shell-state';
  import Sidebar from './Sidebar.svelte';
  import Topbar from './Topbar.svelte';

  /** spacious: reading width. dense: wide lists. split: full-bleed panes that manage their own scroll. */
  type Width = 'spacious' | 'dense' | 'split';

  interface Props {
    crumbs: Crumb[];
    width?: Width;
    children: Snippet;
    footer?: Snippet;
  }

  const widths: Record<Width, string> = {
    spacious: 'mx-auto w-full max-w-5xl px-4 py-8 sm:px-8 lg:py-14',
    dense: 'w-full px-4 py-6 sm:px-6 lg:px-8',
    split: 'w-full flex-1',
  };

  const { crumbs, width = 'dense', children, footer }: Props = $props();
  let open = $state(false);

  function onkeydown(event: KeyboardEvent): void {
    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
      event.preventDefault();
      paletteOpen.update((value) => !value);
    }
  }
</script>

<svelte:window {onkeydown} />

<div class="app-ui drawer min-h-screen bg-base-200 lg:drawer-open">
  <input id="workspace-drawer" type="checkbox" class="drawer-toggle" bind:checked={open} aria-label="Open menu" />
  <div class="drawer-content flex min-h-screen min-w-0 flex-col">
    <Topbar {crumbs} />
    <main class="hero-in flex min-h-0 flex-1 flex-col"><div class="flex flex-1 flex-col {widths[width]}">{@render children()}</div></main>
    {#if footer}{@render footer()}{/if}
  </div>
  <div class="drawer-side z-20">
    <label for="workspace-drawer" aria-label="Close menu" class="drawer-overlay"></label>
    <Sidebar />
  </div>
  <CommandPalette />
</div>
