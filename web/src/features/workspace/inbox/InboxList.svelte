<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import StatusDot from '../components/StatusDot.svelte';
  import { findProject, type InboxItem } from '../logic/derive';
  import { acceptTask } from '../logic/tasks-api';
  import { projects, refresh } from '../logic/workspace-store';
  import { timeAgo } from '../logic/time';

  interface Props {
    items: InboxItem[];
    selectedId: string | undefined;
  }

  const { items, selectedId }: Props = $props();
  let selected = $state<string[]>([]);
  let busy = $state(false);
  let message = $state('');
  const eligible = $derived(items.filter((item) => item.task.status === 'review'));
  const eligibleIds = $derived(new Set(eligible.map((item) => item.id)));
  const selectedVisible = $derived(selected.filter((id) => eligibleIds.has(id)));

  $effect(() => {
    const allowed = new Set(eligible.map((item) => item.id));
    const next = selected.filter((id) => allowed.has(id));
    if (next.length !== selected.length) selected = next;
  });

  function toggle(id: string): void {
    selected = selected.includes(id) ? selected.filter((item) => item !== id) : [...selected, id];
  }

  async function complete(ids: string[]): Promise<void> {
    if (busy || ids.length === 0) return;
    busy = true;
    message = '';
    const results = await Promise.allSettled(ids.map((id) => acceptTask(id)));
    const succeeded = results.filter((result) => result.status === 'fulfilled').length;
    const failed = results.length - succeeded;
    selected = ids.filter((_, index) => results[index]?.status === 'rejected');
    message = failed === 0
      ? `${String(succeeded)} task${succeeded === 1 ? '' : 's'} marked done.`
      : `${String(succeeded)} marked done. ${String(failed)} could not be completed and remain selected.`;
    await refresh();
    busy = false;
  }
</script>

<div class="border-b border-[var(--brand-border)] px-3 py-2">
  <div class="flex flex-col gap-1.5">
    <div class="flex min-w-0 items-center justify-between gap-2">
      <label class="flex min-h-9 min-w-0 items-center gap-2 text-xs">
        <input class="checkbox checkbox-sm" type="checkbox"
          checked={eligible.length > 0 && selectedVisible.length === eligible.length}
          disabled={eligible.length === 0 || busy}
          aria-label="Select all items eligible to mark done"
          onchange={(event) => { selected = event.currentTarget.checked ? eligible.map((item) => item.id) : []; }} />
        <span class="truncate">Select all ({eligible.length})</span>
      </label>
      <span class="shrink-0 text-xs text-[var(--brand-text-secondary)]" aria-live="polite">{selectedVisible.length} selected</span>
    </div>
    <div class="flex min-w-0 flex-wrap items-center gap-1.5">
      {#if selectedVisible.length > 0}
        <button class="btn btn-primary btn-xs" type="button" disabled={busy}
          onclick={() => void complete(selectedVisible)}>Mark selected done</button>
      {/if}
      {#if eligible.length > 0}
        <button class="btn btn-outline btn-xs" type="button" disabled={busy}
          onclick={() => void complete(eligible.map((item) => item.id))}>Mark all done</button>
      {/if}
    </div>
  </div>
  <p class="mt-1 text-xs text-[var(--brand-text-tertiary)]">Only review-ready tasks can be marked done here. Open blocked or stale items to resolve them.</p>
  {#if message !== ''}<p class="mt-1 text-sm" role="status" aria-live="polite">{message}</p>{/if}
</div>

<ul class="flex flex-col" aria-label="Things waiting for you">
  {#each items as item (item.id)}
    <li class="border-b border-[var(--brand-border)]">
      <div class="flex w-full min-w-0 items-stretch gap-1">
        <label class="flex min-h-16 w-10 shrink-0 items-center justify-center" aria-label="Select {item.task.title} to mark done">
          <input class="checkbox checkbox-sm" type="checkbox" checked={selectedVisible.includes(item.id)}
            disabled={item.task.status !== 'review' || busy} aria-label="Select {item.task.title} to mark done"
            onchange={() => { toggle(item.id); }} />
        </label>
        <a
          href="/app/inbox/{item.id}"
          aria-current={selectedId === item.id ? 'page' : undefined}
          class="flex min-h-16 min-w-0 flex-1 items-start gap-2 py-3 pr-3 hover:bg-base-100/70 {selectedId === item.id ? 'bg-base-100 shadow-[inset_3px_0_0_var(--color-primary)]' : ''}"
        >
          <span class="mt-0.5"><StatusDot status={item.task.status} /></span>
          <span class="flex min-w-0 flex-1 flex-col">
            <span class="truncate text-[15px] font-medium">{item.task.title}</span>
            <span class="truncate text-sm text-[var(--brand-text-secondary)]">{item.headline}</span>
            <span class="mt-0.5 text-xs text-[var(--brand-text-tertiary)]">{findProject($projects, item.task.projectId)?.name} · {timeAgo(item.task.updatedAt)}</span>
          </span>
          <Icon name="chevron-right" class="mt-1 size-4 shrink-0 text-[var(--brand-text-tertiary)]" />
        </a>
      </div>
    </li>
  {/each}
</ul>
