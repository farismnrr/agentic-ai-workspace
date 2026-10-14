<script lang="ts">
  import EmptyState from '../../../shared/ui/EmptyState.svelte';
  import { projectsCopy } from '../content/projects-copy';
  import { statusMeta, statusOrder } from '../content/status';
  import { tasksOf } from '../logic/derive';
  import type { Task, TaskStatus } from '../logic/task-types';
  import { tasks } from '../logic/workspace-store';
  import TaskRow from './TaskRow.svelte';

  interface Props {
    projectId: string;
  }

  type Filter = 'all' | 'waiting' | 'running' | 'done';

  const { projectId }: Props = $props();
  const groups: Record<Filter, (task: Task) => boolean> = {
    all: () => true,
    waiting: (task) => statusMeta[task.status].needsYou,
    running: (task) => ['working', 'planning', 'queued'].includes(task.status),
    done: (task) => task.status === 'done',
  };
  const labels: Record<Filter, string> = { all: 'All', waiting: 'Needs you', running: 'In progress', done: 'Done' };
  const rank = (status: TaskStatus): number => statusOrder.indexOf(status);

  let filter = $state<Filter>('all');
  const all = $derived(tasksOf($tasks, projectId));
  const rows = $derived(all.filter(groups[filter]).sort((a, b) => rank(a.status) - rank(b.status)));
  const counts = $derived(Object.fromEntries((Object.keys(groups) as Filter[]).map((key) => [key, all.filter(groups[key]).length])));
</script>

<div role="tablist" class="flex flex-wrap gap-2" aria-label="Filter tasks">
  {#each Object.keys(groups) as key (key)}
    <button role="tab" aria-selected={filter === key} class="btn btn-sm min-h-9 rounded-full {filter === key ? 'btn-primary' : 'btn-outline border-[var(--brand-border)]'}" onclick={() => (filter = key as Filter)}>
      {labels[key as Filter]} <span class="opacity-70">{counts[key]}</span>
    </button>
  {/each}
</div>
<div class="mt-3 overflow-hidden card border border-[var(--brand-border)] bg-base-100">
  {#each rows as task (task.id)}<TaskRow {task} />{:else}
    {#if all.length === 0}
      <EmptyState compact icon="check-square" title={projectsCopy.noTasks.title} body={projectsCopy.noTasks.body}>
        <a class="btn btn-primary btn-sm rounded-lg" href="/app">New task</a>
      </EmptyState>
    {:else}
      <EmptyState compact icon="search" title="Nothing in “{labels[filter]}”" body="Pick another filter to see the rest of this project's tasks." />
    {/if}
  {/each}
</div>
