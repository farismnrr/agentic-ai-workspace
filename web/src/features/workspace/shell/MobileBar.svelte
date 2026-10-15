<script lang="ts">
  import { route } from '../../../app/router';
  import Icon from '../../../shared/ui/Icon.svelte';
  import type { IconName } from '../../../shared/ui/icon-names';
  import { inboxItems } from '../logic/derive';
  import { runningSites } from '../logic/sites';
  import { sites, tasks } from '../logic/workspace-store';

  const name = $derived($route.name);
  const waiting = $derived(inboxItems($tasks).length);
  const live = $derived(runningSites($sites).length);

  interface Entry {
    href: string;
    icon: IconName;
    label: string;
    active: boolean;
    badge: number;
  }

  const entries = $derived<Entry[]>([
    { href: '/app', icon: 'home', label: 'Home', active: name === 'home', badge: 0 },
    { href: '/app/inbox', icon: 'inbox', label: 'Inbox', active: name === 'inbox', badge: waiting },
    { href: '/app/projects', icon: 'folder', label: 'Projects', active: name === 'projects' || name === 'project', badge: 0 },
    { href: '/app/sites', icon: 'globe', label: 'Sites', active: name === 'sites', badge: live },
    { href: '/app/settings', icon: 'settings', label: 'Settings', active: name === 'settings', badge: 0 },
  ]);
</script>

<nav class="dock dock-sm z-10 border-t border-[var(--brand-border)] bg-[var(--brand-panel)] md:hidden" aria-label="Main">
  {#each entries as entry (entry.href)}
    <a href={entry.href} class={entry.active ? 'dock-active' : ''} aria-current={entry.active ? 'page' : undefined}>
      <span class="indicator">
        <Icon name={entry.icon} class="size-[1.2em]" />
        {#if entry.badge > 0}<span class="indicator-item badge badge-primary badge-xs">{entry.badge}</span>{/if}
      </span>
      <span class="dock-label">{entry.label}</span>
    </a>
  {/each}
</nav>
