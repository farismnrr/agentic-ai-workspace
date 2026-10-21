<script lang="ts">
  import { route } from '../../../app/router';
  import Icon from '../../../shared/ui/Icon.svelte';
  import type { Task } from '../logic/task-types';
  import { setTaskPinned } from '../logic/workspace-store';
  import { timeAgo } from '../logic/time';
  import { dotTone } from '../components/tone';
  import { statusMeta } from '../content/status';

  interface Props {
    task: Task;
    projectName?: string;
  }

  let { task, projectName }: Props = $props();
  let saving = $state(false);
  let error = $state('');
  const active = $derived($route.name === 'task' && $route.taskId === task.id);
  const pinned = $derived(task.pinnedAt !== null);
  const pinLabel = $derived(pinned ? 'Unpin task' : 'Pin task');
  const meta = $derived(statusMeta[task.status]);

  async function togglePin(): Promise<void> {
    saving = true;
    error = '';
    try {
      await setTaskPinned(task.id, task.pinnedAt === null);
    } catch {
      error = 'Could not update pin. Try again.';
    } finally {
      saving = false;
    }
  }
</script>

<li class="group flex items-center gap-1 rounded-lg hover:bg-base-100/70 {active ? 'bg-base-300/60 font-medium' : ''}">
  <a
    href="/app/projects/{task.projectId}/tasks/{task.id}"
    aria-current={active ? 'page' : undefined}
    class="flex min-w-0 flex-1 items-center gap-2 px-2 py-1.5"
  >
    <span class="grid size-2 shrink-0 place-items-center">
      {#if meta.needsYou || task.status === 'working'}<span class="status {dotTone[meta.tone]}"></span>{/if}
    </span>
    <span class="min-w-0 flex-1">
      <span class="block truncate">{task.title}</span>
      {#if projectName}<span class="block truncate text-xs font-normal text-[var(--brand-text-tertiary)]">{projectName}</span>{/if}
    </span>
    <span class="shrink-0 text-xs font-normal text-[var(--brand-text-tertiary)]">
      {timeAgo(task.pinnedAt ?? task.updatedAt).replace(' ago', '')}
    </span>
  </a>
  <button
    type="button"
    class="btn btn-ghost btn-circle btn-xs mr-1 shrink-0"
    aria-label={pinLabel}
    title={pinLabel}
    aria-pressed={pinned}
    disabled={saving}
    onclick={(event) => { event.preventDefault(); event.stopPropagation(); void togglePin(); }}
  >
    <Icon name={pinned ? 'pin-filled' : 'pin'} class="size-4 {pinned ? 'text-primary' : ''}" />
  </button>
  {#if error}<span class="alert alert-error alert-soft basis-full px-2 py-1 text-xs" role="alert">{error}</span>{/if}
</li>
