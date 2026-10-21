<script lang="ts">
  import { refresh, removeTasks } from '../logic/workspace-store';
  import { changeTasks, type TaskBatchAction, type TaskBatchResult } from '../logic/tasks-api';
  import type { Task } from '../logic/task-types';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import StatusBadge from '../components/StatusBadge.svelte';
  import StatusDot from '../components/StatusDot.svelte';
  import { timeAgo } from '../logic/time';

  interface Props {
    tasks: Task[];
    archived?: boolean;
    onchanged?: (action: TaskBatchAction, ids: string[]) => void;
  }

  const { tasks, archived = false, onchanged }: Props = $props();
  let selected = $state<string[]>([]);
  let selectAll: HTMLInputElement;
  let confirming = $state<'archive' | 'delete' | undefined>();
  let message = $state('');
  let busy = $state(false);
  const selectedTasks = $derived(tasks.filter((task) => selected.includes(task.id)));
  const runningCount = $derived(selectedTasks.filter((task) => ['queued', 'planning', 'working'].includes(task.status)).length);

  $effect(() => {
    const shown = new Set(tasks.map((task) => task.id));
    selected = selected.filter((id) => shown.has(id));
  });
  $effect(() => {
    selectAll.indeterminate = selected.length > 0 && selected.length < tasks.length;
  });

  function toggle(id: string): void {
    selected = selected.includes(id) ? selected.filter((item) => item !== id) : [...selected, id];
  }

  function request(action: 'archive' | 'delete'): void {
    if (action === 'delete' || runningCount > 0) confirming = action;
    else void apply('archive');
  }

  async function apply(action: TaskBatchAction): Promise<void> {
    confirming = undefined;
    busy = true;
    message = '';
    const ids = [...selected];
    try {
      const results = await changeTasks(action, ids);
      finish(action, ids, results);
    } catch {
      message = 'The tasks could not be updated. Try again.';
    } finally {
      busy = false;
    }
  }

  function finish(action: TaskBatchAction, ids: string[], results: TaskBatchResult[]): void {
    const succeeded = results.filter((result) => result.ok).map((result) => result.id);
    const failed = new Set(results.filter((result) => !result.ok).map((result) => result.id));
    selected = ids.filter((id) => failed.has(id));
    if (action === 'archive' || action === 'delete') removeTasks(succeeded);
    if (action === 'restore') void refresh();
    onchanged?.(action, succeeded);
    const names = tasks.filter((task) => failed.has(task.id)).map((task) => task.title);
    message = failed.size === 0
      ? `${String(succeeded.length)} task${succeeded.length === 1 ? '' : 's'} ${action === 'delete' ? 'deleted' : `${action}d`}.`
      : `Could not update: ${names.join(', ') || `${String(failed.size)} task${failed.size === 1 ? '' : 's'}`}. Those tasks remain selected.`;
  }
</script>

<div class="mt-3">
  <div class="flex flex-wrap items-center gap-3 border-b border-[var(--brand-border)] px-3 py-2">
    <label class="flex items-center gap-2 text-sm">
      <input bind:this={selectAll} class="checkbox checkbox-sm" type="checkbox"
        checked={tasks.length > 0 && selected.length === tasks.length} disabled={tasks.length === 0 || busy}
        onchange={(event) => { selected = event.currentTarget.checked ? tasks.map((task) => task.id) : []; }} />
      Select all shown
    </label>
    {#if selected.length > 0}
      <span class="text-sm" aria-live="polite">{selected.length} selected</span>
      <div class="ml-auto flex gap-2">
        {#if archived}
          <button class="btn btn-outline btn-sm" type="button" disabled={busy} onclick={() => void apply('restore')}>Restore</button>
        {:else}
          <button class="btn btn-outline btn-sm" type="button" disabled={busy} onclick={() => { request('archive'); }}>Archive</button>
        {/if}
        <button class="btn btn-error btn-outline btn-sm" type="button" disabled={busy} onclick={() => { request('delete'); }}>Delete permanently</button>
      </div>
    {/if}
  </div>
  <p class="min-h-5 px-3 pt-1 text-sm" aria-live="polite">{message}</p>
  {#each tasks as task (task.id)}
    <div class="flex min-h-14 items-center gap-3 border-b border-[var(--brand-border)] px-3 py-2 last:border-b-0 hover:bg-base-200/60">
      <input class="checkbox checkbox-sm" type="checkbox" checked={selected.includes(task.id)} disabled={busy}
        aria-label="Select {task.title}" onchange={() => { toggle(task.id); }} />
      <a href="/app/projects/{task.projectId}/tasks/{task.id}" class="flex min-w-0 flex-1 items-center gap-3 focus-visible:outline-primary">
        <StatusDot status={task.status} />
        <span class="flex min-w-0 flex-1 flex-col">
          <span class="truncate font-medium">{task.title}</span>
          <span class="truncate text-sm text-[var(--brand-text-secondary)]">{task.summary === '' ? task.model : task.summary}</span>
        </span>
        <span class="hidden md:block"><StatusBadge status={task.status} /></span>
        <span class="hidden w-24 text-right text-sm text-[var(--brand-text-tertiary)] sm:block">{timeAgo(task.updatedAt)}</span>
      </a>
      {#if archived}
        <button class="btn btn-outline btn-xs" type="button" disabled={busy}
          onclick={() => { selected = [task.id]; void apply('restore'); }}>Restore</button>
        <button class="btn btn-error btn-outline btn-xs" type="button" disabled={busy}
          onclick={() => { selected = [task.id]; request('delete'); }}>Delete permanently</button>
      {/if}
    </div>
  {:else}
    <p class="px-3 py-4 text-sm text-[var(--brand-text-secondary)]">{archived ? 'Nothing archived' : 'No tasks yet.'}</p>
  {/each}
</div>

<ConfirmModal open={confirming !== undefined}
  title={confirming === 'delete' ? `Delete ${String(selected.length)} tasks permanently?` : `Archive ${String(selected.length)} tasks?`}
  body={confirming === 'delete' ? 'This cannot be undone.' : 'The selected tasks will move to Archive.'}
  warning={runningCount > 0 ? `${String(runningCount)} tasks are still running and will be stopped.` : ''}
  confirmLabel={confirming === 'delete' ? 'Delete permanently' : 'Archive'} cancelLabel="Cancel"
  oncancel={() => (confirming = undefined)} onconfirm={() => void apply(confirming ?? 'archive')} />
