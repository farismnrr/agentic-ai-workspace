<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '../../../shared/api/http';
  import { activityCopy } from '../content/connected';
  import { loadActivity, undoActivity, type ActivityEntry, type ActivityOutcome } from '../logic/connectors-api';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import SettingsSection from './parts/SettingsSection.svelte';

  let entries: ActivityEntry[] = $state([]);
  let loading = $state(true);
  let failed = $state(false);
  let pending: ActivityEntry | undefined = $state();
  let problem = $state('');

  const badge: Record<ActivityOutcome, string> = { ok: 'badge-success', failed: 'badge-error', off: 'badge-warning', forbidden: 'badge-warning' };
  const when = (iso: string): string => new Date(iso).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' });

  async function refresh(): Promise<void> {
    try {
      entries = await loadActivity();
      failed = false;
    } catch {
      failed = true;
    } finally {
      loading = false;
    }
  }

  function explain(error: unknown): string {
    const known: Record<string, string> = activityCopy.undoErrors;
    return (error instanceof ApiError ? known[error.code] : undefined) ?? activityCopy.undoFallback;
  }

  async function confirmUndo(): Promise<void> {
    if (pending === undefined) return;
    const target = pending;
    pending = undefined;
    problem = '';
    try {
      await undoActivity(target.id);
    } catch (error) {
      problem = explain(error);
    }
    await refresh();
  }

  onMount(refresh);
</script>

<SettingsSection title="Activity" description={activityCopy.intro}>
  {#if problem !== ''}<p class="py-3 text-error" role="alert">{problem}</p>{/if}
  {#if loading}
    <p class="py-3"><span class="loading loading-spinner loading-sm" aria-label="Loading"></span></p>
  {:else if failed}
    <p class="py-3 text-error" role="alert">{activityCopy.failed}</p>
  {:else if entries.length === 0}
    <p class="py-3 text-[var(--brand-text-secondary)]">{activityCopy.empty}</p>
  {:else}
    {#each entries as entry (entry.id)}
      <div class="flex flex-wrap items-center justify-between gap-3 py-2.5">
        <div class="min-w-0 flex-1">
          <p class="truncate font-medium">{entry.summary}</p>
          <p class="text-xs text-[var(--brand-text-tertiary)]">{when(entry.at)}{entry.detail === '' ? '' : ` · ${entry.detail}`}</p>
        </div>
        <span class="badge badge-sm {badge[entry.outcome]}">{activityCopy.outcomes[entry.outcome]}</span>
        {#if entry.undone}
          <span class="badge badge-ghost badge-sm">Undone</span>
        {:else if entry.canUndo}
          <button class="btn btn-sm" type="button" onclick={() => (pending = entry)}>Undo</button>
        {/if}
      </div>
    {/each}
  {/if}
</SettingsSection>
<ConfirmModal
  open={pending !== undefined}
  title="Undo this change?"
  body={pending === undefined ? '' : `${pending.summary} will be put back the way it was before.`}
  confirmLabel="Undo change"
  cancelLabel="Keep it"
  oncancel={() => (pending = undefined)}
  onconfirm={confirmUndo}
/>
