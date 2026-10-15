<script lang="ts">
  import StatusBadge from '../components/StatusBadge.svelte';
  import type { Task } from '../logic/task-types';
  import ContextMeter from './ContextMeter.svelte';
  import { effortLabel } from '../logic/effort';
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
  <div class="flex flex-wrap items-center gap-x-3 gap-y-1.5 py-2 sm:min-h-12 sm:flex-nowrap sm:py-0">
    <h1 class="line-clamp-2 w-full min-w-0 text-[15px] font-semibold leading-snug sm:line-clamp-1 sm:w-auto">{task.title}</h1>
    <StatusBadge status={task.status} />
    <p class="hidden flex-1 truncate text-xs text-[var(--brand-text-secondary)] md:block">Started {timeAgo(task.createdAt)} · {task.model}{task.effort === '' ? '' : ` · ${effortLabel(task.effort)} effort`}</p>
    <span class="flex-1 md:hidden"></span>
    <ContextMeter used={task.contextUsed} window={task.contextWindow} />
    {#if running}<button class="btn btn-outline btn-sm" type="button" onclick={onstop}>Stop</button>{/if}
    {#if task.status === 'failed'}<button class="btn btn-primary btn-sm" type="button" onclick={onretry}>Continue</button>{/if}
  </div>
  <div class="flex pb-2 sm:hidden"><ContextMeter used={task.contextUsed} window={task.contextWindow} phone /></div>
</header>
