<script lang="ts">
  import type { Snippet } from 'svelte';
  import { onMount } from 'svelte';
  import { startNotifications } from '../logic/notify';
  import { watch } from '../logic/workspace-store';
  import { route } from '../../../app/router';
  import CommandPalette from './CommandPalette.svelte';
  import MobileBar from './MobileBar.svelte';
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
  // A conversation needs the whole phone screen for its composer, so the bar steps aside there.
  const bar = $derived($route.name !== 'task');

  onMount(() => {
    const stopWatching = watch();
    const stopNotifying = startNotifications();
    return () => {
      stopWatching();
      stopNotifying();
    };
  });

  function onkeydown(event: KeyboardEvent): void {
    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
      event.preventDefault();
      paletteOpen.update((value) => !value);
    }
  }
</script>

<svelte:window {onkeydown} />

<div class="app-ui drawer min-h-dvh bg-base-200 lg:drawer-open">
  <input id="workspace-drawer" type="checkbox" class="drawer-toggle" bind:checked={open} aria-label="Open menu" />
  <div class="drawer-content flex min-h-dvh min-w-0 flex-col {bar ? 'pb-16 md:pb-0' : ''}">
    <Topbar {crumbs} />
    <main class="hero-in flex min-h-0 flex-1 flex-col"><div class="flex flex-1 flex-col {widths[width]}">{@render children()}</div></main>
    {#if footer}{@render footer()}{/if}
    {#if bar}<MobileBar />{/if}
  </div>
  <div class="drawer-side z-20">
    <label for="workspace-drawer" aria-label="Close menu" class="drawer-overlay"></label>
    <Sidebar />
  </div>
  <CommandPalette />
</div>
