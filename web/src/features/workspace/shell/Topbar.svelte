<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import { inboxCount } from '../content/stats';
  import { paletteOpen, sidebarCollapsed, type Crumb } from './shell-state';

  interface Props {
    crumbs: Crumb[];
  }

  const { crumbs }: Props = $props();
  const waiting = inboxCount();
</script>

<header class="sticky top-0 z-10 flex h-12 items-center gap-2 border-b border-[var(--brand-border)] bg-base-200/90 px-3 backdrop-blur">
  <label for="workspace-drawer" class="btn btn-ghost btn-sm btn-square lg:hidden" aria-label="Open menu"><Icon name="menu" /></label>
  <button class="btn btn-ghost btn-sm btn-square hidden lg:inline-flex" type="button" aria-label="Collapse or expand the sidebar" onclick={() => { sidebarCollapsed.update((value) => !value); }}>
    <Icon name="sidebar" class="size-[18px]" />
  </button>
  <nav aria-label="Breadcrumb" class="flex min-w-0 flex-1 items-center gap-1.5 text-sm">
    {#each crumbs as crumb, index (crumb.label)}
      {#if index > 0}<span class="text-[var(--brand-text-tertiary)]" aria-hidden="true">/</span>{/if}
      {#if crumb.href !== undefined}
        <a class="truncate text-[var(--brand-text-secondary)] hover:text-base-content" href={crumb.href}>{crumb.label}</a>
      {:else}
        <span class="truncate font-medium" aria-current="page">{crumb.label}</span>
      {/if}
    {/each}
  </nav>
  <button class="btn btn-sm hidden min-h-9 gap-2 rounded-full border-[var(--brand-border)] bg-base-100 font-normal text-[var(--brand-text-secondary)] sm:inline-flex" type="button" onclick={() => { paletteOpen.set(true); }}>
    <Icon name="search" class="size-4" /> Search <kbd class="kbd kbd-sm">Ctrl K</kbd>
  </button>
  <button class="btn btn-ghost btn-sm btn-square sm:hidden" type="button" aria-label="Search" onclick={() => { paletteOpen.set(true); }}><Icon name="search" /></button>
  <a class="btn btn-ghost btn-sm btn-square indicator" href="/app/inbox" aria-label="Inbox, {waiting} waiting for you">
    <Icon name="bell" class="size-[18px]" />
    {#if waiting > 0}<span class="indicator-item badge badge-primary badge-xs">{waiting}</span>{/if}
  </a>
</header>
