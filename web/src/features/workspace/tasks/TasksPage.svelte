<script lang="ts">
  import AppShell from '../shell/AppShell.svelte';
  import EmptyState from '../../../shared/ui/EmptyState.svelte';
  import SkeletonList from '../../../shared/ui/SkeletonList.svelte';
  import TaskSelection from './TaskSelection.svelte';
  import { loaded, tasks } from '../logic/workspace-store';

  const crumbs = [{ label: 'Tasks', href: '/app/tasks' }];
</script>

<svelte:head><title>Tasks · Masih Awam</title></svelte:head>
<AppShell {crumbs} width="dense">
  <header class="mb-4">
    <h1 class="text-xl font-semibold">Tasks</h1>
    <p class="mt-1 text-sm text-[var(--brand-text-secondary)]">All your active work in one place.</p>
  </header>
  <section class="overflow-hidden rounded-xl border border-[var(--brand-border)] bg-base-100" aria-label="Active tasks">
  {#if !$loaded}
    <div class="p-3"><SkeletonList rows={5} /></div>
  {:else if $tasks.length === 0}
    <EmptyState compact icon="check-square" title="No active tasks" body="Start a task from Home and it will appear here." />
  {:else}
    <TaskSelection tasks={$tasks} />
  {/if}
  </section>
</AppShell>
