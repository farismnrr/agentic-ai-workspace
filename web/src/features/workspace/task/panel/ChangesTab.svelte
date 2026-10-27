<script lang="ts">
  import { onDestroy, untrack } from 'svelte';
  import EmptyState from '../../../../shared/ui/EmptyState.svelte';
  import Icon from '../../../../shared/ui/Icon.svelte';
  import ChangedFile from '../../changes/ChangedFile.svelte';
  import { changesCopy } from '../../content/changes-copy';
  import { createFileChanges } from '../../logic/changes-store';
  import type { Change } from '../../logic/task-types';
  import EvidenceBadge from '../thread/EvidenceBadge.svelte';

  interface Props {
    taskId: string;
    projectId: string;
    changes: Change[];
    onundo: (eventId: string) => void;
  }

  const { taskId, projectId, changes, onundo }: Props = $props();
  const store = untrack(() => createFileChanges(taskId, projectId));
  const files = store.files;
  const loaded = store.loaded;
  let opened = $state<Record<string, boolean>>({});

  // Each step of the AI adds a change, which may not have reached the files stream yet.
  $effect(() => {
    const steps = changes.length;
    untrack(() => { if (steps >= 0) void store.refresh(); });
  });

  onDestroy(store.stop);
  const isOpen = (path: string, index: number): boolean => opened[path] ?? index === 0;
</script>

<div class="flex flex-col gap-3">
  {#if $files.length === 0 && changes.length === 0}
    <EmptyState compact bordered icon="pencil" title="No changes yet" body={changesCopy.empty} />
  {:else}
    {#if $files.length > 0}
      <p class="text-sm font-medium">{$files.length} {$files.length === 1 ? 'file' : 'files'} changed</p>
      {#each $files as change, index (change.path)}
        <ChangedFile {change} open={isOpen(change.path, index)} ontoggle={() => { opened = { ...opened, [change.path]: !isOpen(change.path, index) }; }} />
      {/each}
    {:else if !$loaded}
      <div class="grid place-items-center py-6"><span class="loading loading-spinner"></span></div>
    {/if}

    <details class="rounded-lg border border-[var(--brand-border)] bg-base-100 p-3" open={$files.length === 0}>
      <summary class="cursor-pointer text-sm font-medium">Steps, with Undo ({changes.length})</summary>
      <ul class="mt-2 flex flex-col gap-2">
        {#each changes as change (change.id)}
          <li class="rounded-lg border border-[var(--brand-border)] p-3 text-sm">
            <p class="flex flex-wrap items-center gap-2 font-medium"><Icon name="pencil" class="size-4 shrink-0 text-secondary" />{change.text}<EvidenceBadge work={change.work} exitCode={change.exitCode} /></p>
            <ul class="mt-2 flex flex-wrap gap-1">
              {#each change.files as file, index (index)}<li><code class="rounded bg-base-200 px-1.5 py-0.5 text-xs">{file}</code></li>{/each}
            </ul>
            <div class="mt-2">
              {#if change.undone}<span class="badge badge-ghost badge-sm">Undone</span>
              {:else if change.canUndo}<button class="btn btn-outline btn-xs" type="button" onclick={() => { onundo(change.id); }}>Undo this change</button>{/if}
            </div>
          </li>
        {/each}
      </ul>
    </details>
  {/if}
</div>
