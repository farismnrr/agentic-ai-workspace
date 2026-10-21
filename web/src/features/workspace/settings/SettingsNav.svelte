<script lang="ts">
  import type { SettingsTab } from '../../../app/router';
  import Icon from '../../../shared/ui/Icon.svelte';
  import { visibleNavGroups } from './settings-nav';

  interface Props {
    tab: SettingsTab;
    admin: boolean;
  }

  const { tab, admin }: Props = $props();
  const groups = $derived(visibleNavGroups(admin));
</script>

<nav aria-label="Settings sections" class="flex gap-1 overflow-x-auto md:flex-col md:overflow-visible">
  {#each groups as group (group.label)}
    <p class="hidden px-2 pb-1 pt-3 text-xs text-[var(--brand-text-tertiary)] first:pt-0 md:block">{group.label}</p>
    {#each group.items as item (item.id)}
      <a
        href="/app/settings/{item.id}"
        aria-current={tab === item.id ? 'page' : undefined}
        class="flex h-8 shrink-0 items-center gap-2 rounded-lg px-2 text-left transition-colors hover:bg-base-100/70 {tab === item.id ? 'bg-base-300/60 font-medium' : 'text-[var(--brand-text-secondary)]'}"
      >
        <Icon name={item.icon} class="size-4 shrink-0" />{item.label}
      </a>
    {/each}
  {/each}
</nav>
