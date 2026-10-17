<script lang="ts">
  import { onDestroy, untrack } from 'svelte';
  import EmptyState from '../../../../shared/ui/EmptyState.svelte';
  import Icon from '../../../../shared/ui/Icon.svelte';
  import ChangedFile from '../../changes/ChangedFile.svelte';
  import { createFileChanges } from '../../logic/changes-store';
  import type { Change } from '../../logic/task-types';

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
  let wide = $state(false);
  let layout = $state('');
  let opened = $state<Record<string, boolean>>({});
  let box = $state<HTMLDivElement>();

  // Each step of the AI adds a change, which may not have reached the files stream yet.
  $effect(() => {
    const steps = changes.length;
    untrack(() => { if (steps >= 0) void store.refresh(); });
  });

  // A wide panel shows old and new side by side; a narrow one stacks them. The person can choose either.
  $effect(() => {
    if (box === undefined) return;
    const watch = new ResizeObserver(([entry]) => { wide = (entry?.contentRect.width ?? 0) >= 720; });
    watch.observe(box);
    return () => { watch.disconnect(); };
  });

  onDestroy(store.stop);
  const split = $derived(layout === '' ? wide : layout === 'split');
  const isOpen = (path: string, index: number): boolean => opened[path] ?? index === 0;
</script>

<div bind:this={box} class="flex flex-col gap-3">
  {#if $files.length === 0 && changes.length === 0}
    <EmptyState compact bordered icon="pencil" title="No changes yet" body="Edits the AI makes to this project show up here, file by file, and you can undo each step." />
  {:else}
    {#if $files.length > 0}
      <div class="flex flex-wrap items-center gap-2">
        <p class="flex-1 text-sm font-medium">{$files.length} {$files.length === 1 ? 'file' : 'files'} changed</p>
        <div class="join" role="group" aria-label="How to compare">
          <button type="button" class="btn btn-xs join-item {split ? '' : 'btn-primary'}" aria-pressed={!split} onclick={() => { layout = 'unified'; }}>Unified</button>
          <button type="button" class="btn btn-xs join-item {split ? 'btn-primary' : ''}" aria-pressed={split} onclick={() => { layout = 'split'; }}>Split</button>
        </div>
      </div>
      {#each $files as change, index (change.path)}
        <ChangedFile {change} {split} open={isOpen(change.path, index)} ontoggle={() => { opened = { ...opened, [change.path]: !isOpen(change.path, index) }; }} />
      {/each}
    {:else if !$loaded}
      <div class="grid place-items-center py-6"><span class="loading loading-spinner"></span></div>
    {/if}

    <details class="rounded-lg border border-[var(--brand-border)] bg-base-100 p-3" open={$files.length === 0}>
      <summary class="cursor-pointer text-sm font-medium">Steps, with Undo ({changes.length})</summary>
      <ul class="mt-2 flex flex-col gap-2">
        {#each changes as change (change.id)}
          <li class="rounded-lg border border-[var(--brand-border)] p-3 text-sm">
            <p class="flex items-center gap-2 font-medium"><Icon name="pencil" class="size-4 text-secondary" />{change.text}</p>
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
