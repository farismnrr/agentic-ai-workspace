<script lang="ts">
  import type { IconName } from '../../../shared/ui/icon-names';
  import NavItemBody from './NavItemBody.svelte';

  interface Props {
    icon: IconName;
    label: string;
    collapsed: boolean;
    href?: string | undefined;
    onclick?: (() => void) | undefined;
    active?: boolean;
    badge?: number | undefined;
    hint?: string | undefined;
  }

  const { icon, label, collapsed, href, onclick, active = false, badge, hint }: Props = $props();
  const row = $derived(
    `flex w-full items-center gap-2.5 rounded-lg px-2 py-1.5 text-left transition-colors hover:bg-base-100/70 ${active ? 'bg-base-300/60 font-medium' : ''}`,
  );
</script>

{#if href !== undefined}
  <a {href} title={label} aria-current={active ? 'page' : undefined} class={row}><NavItemBody {icon} {label} {collapsed} {badge} {hint} /></a>
{:else}
  <button type="button" title={label} {onclick} class={row}><NavItemBody {icon} {label} {collapsed} {badge} {hint} /></button>
{/if}
