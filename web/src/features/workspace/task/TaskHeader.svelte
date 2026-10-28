<script lang="ts">
  import StatusBadge from '../components/StatusBadge.svelte';
  import type { Task } from '../logic/task-types';
  import ContextMeter from './ContextMeter.svelte';
  import { effortLabel } from '../logic/effort';
  import { isExternalTask } from '../logic/task-links';
  import { timeAgo } from '../logic/time';
  import { workStatusPresentation } from '../logic/work-status';
  import { statusMeta } from '../content/status';

  interface Props {
    task: Task;
    archived?: boolean;
    onstop: () => void;
    onconfirm: () => void;
    onretry: () => void;
  }

  const { task, archived = false, onstop, onconfirm, onretry }: Props = $props();
  const external = $derived(isExternalTask(task.source, task.workId ?? ''));
  const workStatus = $derived(workStatusPresentation(task.status, task.work));
  const statusLabel = $derived(workStatus.label || statusMeta[task.status].label);
</script>

<header class="border-b border-[var(--brand-border)] bg-base-100 px-4 sm:px-6">
  <div class="flex flex-wrap items-center gap-x-3 gap-y-1.5 py-2 sm:min-h-12 sm:flex-nowrap sm:py-0">
    <h1 class="line-clamp-2 w-full min-w-0 text-[15px] font-semibold leading-snug sm:line-clamp-1 sm:w-auto">{task.title}</h1>
    <StatusBadge status={workStatus.iconStatus} label={statusLabel} />
    {#if task.work?.state === 'blocked'}<span class="badge badge-warning badge-sm">Blocked</span>{/if}
    {#if task.team}<span class="badge badge-soft badge-sm">Team</span>{/if}
    <p class="hidden min-w-0 flex-1 truncate text-xs text-[var(--brand-text-secondary)] md:block">
      {#if external}External assistant · {task.work?.actor ?? task.source}
      {:else}Started {timeAgo(task.createdAt)} · {task.model}{task.effort === '' ? '' : ` · ${effortLabel(task.effort)} effort`}{/if}
      {#if workStatus.hint !== ''} · {workStatus.hint}{/if}
    </p>
    <span class="flex-1 md:hidden"></span>
    <ContextMeter used={task.contextUsed} window={task.contextWindow} />
    {#if !archived && workStatus.canStop}<button class="btn btn-outline btn-sm" type="button" aria-label="Stop work" onclick={onstop}>Stop</button>{/if}
    {#if !archived && workStatus.canConfirm}<button class="btn btn-primary btn-sm" type="button" onclick={onconfirm}>Mark complete</button>{/if}
    {#if !archived && !external && task.status === 'failed'}<button class="btn btn-primary btn-sm" type="button" onclick={onretry}>Continue</button>{/if}
  </div>
  {#if external}
    <p class="pb-2 text-xs text-[var(--brand-text-secondary)] md:hidden">
      External assistant · {task.work?.actor ?? task.source}
      {#if workStatus.hint !== ''} · {workStatus.hint}{/if}
    </p>
  {/if}
  <div class="flex pb-2 sm:hidden"><ContextMeter used={task.contextUsed} window={task.contextWindow} phone /></div>
</header>
