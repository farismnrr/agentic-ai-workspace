<script lang="ts">
  import { onMount } from 'svelte';
  import EmptyState from '../../../shared/ui/EmptyState.svelte';
  import StatusBadge from '../components/StatusBadge.svelte';
  import StatusDot from '../components/StatusDot.svelte';
  import { statusMeta, statusOrder } from '../content/status';
  import { findProject } from '../logic/derive';
  import { loadTasks } from '../logic/tasks-api';
  import type { Task } from '../logic/task-types';
  import { projects } from '../logic/workspace-store';
  import SettingsSection from './parts/SettingsSection.svelte';

  type SortBy = 'recent' | 'title' | 'status';
  let tasks: Task[] = $state([]);
  let loading = $state(true);
  let failed = $state(false);
  let status = $state('all');
  let project = $state('all');
  let sortBy = $state<SortBy>('recent');

  const visible = $derived(tasks
    .filter((task) => status === 'all' || task.status === status)
    .filter((task) => project === 'all' || task.projectId === project)
    .sort((a, b) => {
      if (sortBy === 'title') return a.title.localeCompare(b.title);
      if (sortBy === 'status') return statusOrder.indexOf(a.status) - statusOrder.indexOf(b.status);
      return b.updatedAt.localeCompare(a.updatedAt);
    }));

  async function refresh(): Promise<void> {
    loading = true;
    failed = false;
    try {
      tasks = await loadTasks();
    } catch {
      failed = true;
    } finally {
      loading = false;
    }
  }

  onMount(() => void refresh());
</script>

<SettingsSection title="Tasks" description="Find work across your projects and open its details.">
  <div class="flex flex-wrap gap-2 py-3" aria-label="Filter and sort tasks">
    <label class="form-control min-w-36 flex-1">
      <span class="label-text mb-1 text-xs">Status</span>
      <select class="select select-bordered select-sm w-full" bind:value={status} aria-label="Filter tasks by status">
        <option value="all">All statuses</option>
        {#each statusOrder as value (value)}<option value={value}>{statusMeta[value].label}</option>{/each}
      </select>
    </label>
    <label class="form-control min-w-36 flex-1">
      <span class="label-text mb-1 text-xs">Project</span>
      <select class="select select-bordered select-sm w-full" bind:value={project} aria-label="Filter tasks by project">
        <option value="all">All projects</option>
        {#each $projects as item (item.id)}<option value={item.id}>{item.name}</option>{/each}
      </select>
    </label>
    <label class="form-control min-w-36 flex-1">
      <span class="label-text mb-1 text-xs">Sort by</span>
      <select class="select select-bordered select-sm w-full" bind:value={sortBy} aria-label="Sort tasks">
        <option value="recent">Recently updated</option>
        <option value="title">Title</option>
        <option value="status">Status</option>
      </select>
    </label>
  </div>
  {#if loading}
    <p class="flex min-h-24 items-center justify-center gap-2 py-4 text-sm text-[var(--brand-text-secondary)]" role="status">
      <span class="loading loading-spinner loading-sm" aria-hidden="true"></span>Loading tasks…
    </p>
  {:else if failed}
    <div class="flex flex-wrap items-center justify-between gap-3 py-4" role="alert">
      <p class="text-sm text-error">Tasks could not be loaded. Check your connection and try again.</p>
      <button class="btn btn-outline btn-sm" type="button" onclick={() => void refresh()}>Try again</button>
    </div>
  {:else if tasks.length === 0}
    <EmptyState compact icon="check-square" title="No tasks yet" body="Start a task from Home, and it will appear here." />
  {:else if visible.length === 0}
    <EmptyState compact icon="search" title="No matching tasks" body="Change the status or project filters to see more tasks." />
  {:else}
    {#each visible as task (task.id)}
      {@const meta = statusMeta[task.status]}
      <a href="/app/projects/{task.projectId}/tasks/{task.id}"
        class="flex min-h-14 min-w-0 items-center gap-3 border-b border-[var(--brand-border)] py-2 last:border-0 hover:bg-base-200/60 focus-visible:outline-primary">
        <StatusDot status={task.status} label={meta.label} />
        <span class="flex min-w-0 flex-1 flex-col">
          <span class="truncate font-medium">{task.title}</span>
          <span class="truncate text-xs text-[var(--brand-text-secondary)]">{findProject($projects, task.projectId)?.name ?? 'Unknown project'}</span>
        </span>
        <span class="hidden sm:block"><StatusBadge status={task.status} /></span>
      </a>
    {/each}
  {/if}
</SettingsSection>
