<script lang="ts">
  import EmptyState from '../../../../shared/ui/EmptyState.svelte';
  import Icon from '../../../../shared/ui/Icon.svelte';
  import type { Change } from '../../logic/task-types';

  interface Props {
    changes: Change[];
  }

  const { changes }: Props = $props();
  const files = $derived([...new Set(changes.filter((change) => !change.undone).flatMap((change) => change.files))]);
</script>

{#if files.length === 0}
  <EmptyState compact bordered icon="file-text" title="No files touched" body="Files this task creates or edits are listed here." />
{:else}
  <ul class="divide-y divide-[var(--brand-border)] rounded-xl border border-[var(--brand-border)] bg-base-100">
    {#each files as file (file)}
      <li class="flex items-center gap-2 px-3 py-2"><Icon name="file-text" class="size-4 shrink-0 text-[var(--brand-text-tertiary)]" /><code class="flex-1 truncate text-xs">{file}</code><span class="badge badge-ghost badge-sm">changed</span></li>
    {/each}
  </ul>
{/if}
