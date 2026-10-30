<script lang="ts">
  import { onDestroy, untrack } from 'svelte';
  import EmptyState from '../../../../shared/ui/EmptyState.svelte';
  import Icon from '../../../../shared/ui/Icon.svelte';
  import { createFileChanges } from '../../logic/changes-store';
  import { createGitStatusStore, gitStatusLabel } from '../../logic/git-status-store';
  import type { Change } from '../../logic/task-types';
  import GitChangeDiff from '../../changes/GitChangeDiff.svelte';
  import EvidenceBadge from '../thread/EvidenceBadge.svelte';

  interface Props {
    taskId: string;
    projectId: string;
    changes: Change[];
    onundo: (eventId: string) => void;
  }

  const { taskId, projectId, changes, onundo }: Props = $props();
  const store = untrack(() => createFileChanges(taskId, projectId));
  const gitStore = untrack(() => createGitStatusStore(projectId));
  const git = gitStore.snapshot;
  const loaded = store.loaded;
  let refreshKey = $state(0);

  $effect(() => {
    const snapshot = $git;
    untrack(() => { if (snapshot !== undefined) refreshKey += 1; });
  });

  // Each step of the AI adds a change, which may not have reached the files stream yet.
  $effect(() => {
    const steps = changes.length;
    untrack(() => { if (steps >= 0) void store.refresh(); });
  });

  onDestroy(() => { store.stop(); gitStore.stop(); });
</script>

<div class="flex flex-col gap-3">
  {#if $git === undefined}
    <div class="grid place-items-center py-6"><span class="loading loading-spinner" aria-label="Loading Git status"></span></div>
  {:else if !$git.available}
    <div class="alert alert-info alert-soft text-sm" role="status">Git tracking is not available for this project.</div>
  {:else if $git.files.length === 0}
    <EmptyState compact icon="pencil" title="No active Git changes" body="Saved files match the current commit. Task steps remain in history below." />
  {:else}
    <p class="text-sm font-medium">Current Git changes · {$git.files.length} {$git.files.length === 1 ? 'file' : 'files'}</p>
    <ul class="flex flex-col divide-y divide-[var(--brand-border)] rounded-lg border border-[var(--brand-border)] bg-base-100">
      {#each $git.files as file (file.path)}
        <li class="flex min-w-0 flex-wrap items-center gap-2 px-3 py-2 text-sm">
          <span class="badge badge-warning badge-sm">{gitStatusLabel(file)}</span>
          <code class="min-w-0 flex-1 truncate">{file.originalPath ? `${file.originalPath} → ${file.path}` : file.path}</code>
          <span class="text-xs text-[var(--brand-text-secondary)]">Staged: {file.index || '—'} · Unstaged: {file.worktree || '—'}</span>
          <GitChangeDiff {projectId} {file} {refreshKey} />
        </li>
      {/each}
    </ul>
  {/if}

  {#if changes.length > 0 || !$loaded}
    <details class="rounded-lg border border-[var(--brand-border)] bg-base-100 p-3" open={$git?.files.length === 0}>
      <summary class="cursor-pointer text-sm font-medium">Task history: steps with Undo ({changes.length})</summary>
      {#if !$loaded}<div class="grid place-items-center py-4"><span class="loading loading-spinner" aria-label="Loading task history"></span></div>{/if}
      <ul class="mt-2 flex flex-col divide-y divide-[var(--brand-border)]">
        {#each changes as change (change.id)}
          <li class="py-3 text-sm">
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
