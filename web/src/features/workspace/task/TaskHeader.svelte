<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import Notice from '../components/Notice.svelte';
  import StatusBadge from '../components/StatusBadge.svelte';
  import type { Task } from '../content/task-types';

  interface Props {
    task: Task;
  }

  const { task }: Props = $props();
  const actions = $derived(task.status === 'working' || task.status === 'planning' ? ['Stop this task', 'Rename', 'Duplicate'] : ['Rename', 'Duplicate', 'Delete']);
  let used = $state(false);
</script>

<header class="border-b border-[var(--brand-border)] bg-base-100 px-4 sm:px-6">
  <div class="flex min-h-12 items-center gap-3">
    <h1 class="min-w-0 truncate text-[15px] font-semibold">{task.title}</h1>
    <StatusBadge status={task.status} />
    <p class="hidden flex-1 truncate text-xs text-[var(--brand-text-secondary)] md:block">
      Started {task.started}{task.duration !== undefined ? ` · took ${task.duration}` : ''} · {task.model}
    </p>
    <details class="dropdown dropdown-end ml-auto">
      <summary class="btn btn-ghost btn-circle btn-sm" aria-label="Task actions"><Icon name="more" /></summary>
      <ul class="dropdown-content menu z-10 w-44 rounded-xl border border-[var(--brand-border)] bg-base-100 p-1.5 shadow-lg">
        {#each actions as action (action)}<li><button type="button" onclick={() => (used = true)}>{action}</button></li>{/each}
      </ul>
    </details>
  </div>
  {#if used}<div class="pb-3"><Notice show /></div>{/if}
</header>
