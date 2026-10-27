<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';
  import type { WorkEventRef } from '../../logic/task-types';

  interface Props {
    work?: WorkEventRef | undefined;
    exitCode?: number | undefined;
  }

  const { work, exitCode }: Props = $props();
  const label = $derived(work?.evidence === 'claimed' ? 'Self-reported'
    : work?.evidence === 'observed' ? 'Seen by the server'
      : work?.evidence === 'ran' && exitCode !== undefined ? `Ran by the server, exit ${String(exitCode)}`
        : '');
  const icon = $derived(work?.evidence === 'claimed' ? 'info'
    : work?.evidence === 'observed' ? 'check-circle' : 'terminal');
</script>

{#if label !== ''}
  <span class="badge badge-outline badge-xs gap-1 align-middle"><Icon name={icon} class="size-3" />{label}</span>
{/if}
