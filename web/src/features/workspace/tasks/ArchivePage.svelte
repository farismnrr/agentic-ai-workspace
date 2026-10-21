<script lang="ts">
  import { onMount } from 'svelte';
  import AppShell from '../shell/AppShell.svelte';
  import EmptyState from '../../../shared/ui/EmptyState.svelte';
  import SkeletonList from '../../../shared/ui/SkeletonList.svelte';
  import TaskSelection from './TaskSelection.svelte';
  import { loadArchivedTasks } from '../logic/tasks-api';
  import type { Task } from '../logic/task-types';

  const crumbs = [{ label: 'Archive', href: '/app/archive' }];
  let archived = $state<Task[]>([]);
  let loaded = $state(false);
  let error = $state('');

  onMount(() => {
    void load();
  });

  async function load(): Promise<void> {
    error = '';
    try {
      archived = await loadArchivedTasks();
    } catch {
      error = 'Archive could not be loaded. Try again.';
    } finally {
      loaded = true;
    }
  }

  function changed(action: 'archive' | 'restore' | 'delete', ids: string[]): void {
    if (action === 'restore' || action === 'delete') archived = archived.filter((task) => !ids.includes(task.id));
  }
</script>

<svelte:head><title>Archive · Masih Awam</title></svelte:head>
<AppShell {crumbs} width="dense">
  <header class="mb-4">
    <div class="flex items-baseline gap-2">
      <h1 class="text-xl font-semibold">Archive</h1>
      {#if loaded}<span class="badge badge-ghost">{archived.length}</span>{/if}
    </div>
    <p class="mt-1 text-sm text-[var(--brand-text-secondary)]">Archived tasks stay here until you restore or delete them.</p>
  </header>
  {#if !loaded}
    <SkeletonList rows={5} />
  {:else if error !== ''}
    <div class="alert alert-error" role="alert">{error} <button class="btn btn-sm" type="button" onclick={() => void load()}>Retry</button></div>
  {:else if archived.length === 0}
    <EmptyState compact icon="box" title="Nothing archived" body="Tasks you archive will appear here." />
  {:else}
    <TaskSelection tasks={archived} archived onchanged={changed} />
  {/if}
</AppShell>
