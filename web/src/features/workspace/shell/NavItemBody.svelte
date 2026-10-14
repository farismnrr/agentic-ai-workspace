<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import type { IconName } from '../../../shared/ui/icon-names';

  interface Props {
    icon: IconName;
    label: string;
    collapsed: boolean;
    badge?: number | undefined;
    hint?: string | undefined;
  }

  const { icon, label, collapsed, badge, hint }: Props = $props();
</script>

<Icon name={icon} class="size-4 shrink-0" />
<span class="flex-1 truncate {collapsed ? 'lg:hidden' : ''}">{label}</span>
{#if hint !== undefined}<kbd class="kbd kbd-xs {collapsed ? 'lg:hidden' : ''}">{hint}</kbd>{/if}
{#if badge !== undefined && badge > 0}<span class="badge badge-sm bg-base-300 {collapsed ? 'lg:hidden' : ''}">{badge}</span>{/if}
