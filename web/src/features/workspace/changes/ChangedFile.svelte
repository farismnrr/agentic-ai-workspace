<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import type { FileChange, FileStatus } from '../logic/changes-api';
  import { diffStats, type DiffStats } from '../logic/diff-stats';
  import DiffView from './DiffView.svelte';

  interface Props {
    change: FileChange;
    split: boolean;
    open: boolean;
    ontoggle: () => void;
  }

  const { change, split, open, ontoggle }: Props = $props();
  let stats = $state<DiffStats>();

  const label: Record<FileStatus, { text: string; tone: string }> = {
    added: { text: 'Added', tone: 'badge-success' },
    modified: { text: 'Modified', tone: 'badge-info' },
    deleted: { text: 'Deleted', tone: 'badge-error' },
    renamed: { text: 'Renamed', tone: 'badge-ghost' },
  };

  $effect(() => {
    if (change.binary) return;
    let gone = false;
    void diffStats(change.before, change.after).then((result) => { if (!gone) stats = result; });
    return () => { gone = true; };
  });
</script>

<section class="overflow-hidden rounded-lg border border-[var(--brand-border)] bg-base-100" aria-label={`Changes in ${change.path}`}>
  <button type="button" class="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-base-200" aria-expanded={open} onclick={ontoggle}>
    <Icon name={open ? 'chevron-down' : 'chevron-right'} class="size-4 shrink-0 text-[var(--brand-text-tertiary)]" />
    <span class="badge badge-sm {label[change.status].tone}">{label[change.status].text}</span>
    <code class="min-w-0 flex-1 truncate text-xs">{change.oldPath !== undefined && change.oldPath !== '' ? `${change.oldPath} → ${change.path}` : change.path}</code>
    {#if stats !== undefined}
      <span class="shrink-0 font-mono text-xs"><span class="text-success">+{stats.added}</span> <span class="text-error">−{stats.removed}</span></span>
    {/if}
  </button>
  {#if open}
    <div class="border-t border-[var(--brand-border)]">
      {#if change.binary}
        <p class="p-3 text-sm text-[var(--brand-text-secondary)]">This file is not text, so its changes cannot be shown.</p>
      {:else if change.status === 'renamed' && change.before === change.after}
        <p class="p-3 text-sm text-[var(--brand-text-secondary)]">Moved without changing its content.</p>
      {:else}
        {#if change.truncated}<p class="px-3 pt-2 text-xs text-[var(--brand-text-secondary)]">This file is large, so only the start is compared.</p>{/if}
        <DiffView path={change.path} before={change.before} after={change.after} {split} />
      {/if}
    </div>
  {/if}
</section>
