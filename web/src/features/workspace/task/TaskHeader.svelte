<script lang="ts">
  import StatusBadge from '../components/StatusBadge.svelte';
  import type { Task } from '../logic/task-types';
  import { timeAgo } from '../logic/time';

  interface Props {
    task: Task;
    onstop: () => void;
    onretry: () => void;
  }

  const { task, onstop, onretry }: Props = $props();
  const running = $derived(['queued', 'planning', 'working'].includes(task.status));
</script>

<header class="border-b border-[var(--brand-border)] bg-base-100 px-4 sm:px-6">
  <div class="flex min-h-12 items-center gap-3">
    <h1 class="min-w-0 truncate text-[15px] font-semibold">{task.title}</h1>
    <StatusBadge status={task.status} />
    <p class="hidden flex-1 truncate text-xs text-[var(--brand-text-secondary)] md:block">Started {timeAgo(task.createdAt)} · {task.model}</p>
    <span class="flex-1 md:hidden"></span>
    {#if running}<button class="btn btn-outline btn-sm" type="button" onclick={onstop}>Stop</button>{/if}
    {#if task.status === 'failed'}<button class="btn btn-primary btn-sm" type="button" onclick={onretry}>Continue</button>{/if}
  </div>
</header>
