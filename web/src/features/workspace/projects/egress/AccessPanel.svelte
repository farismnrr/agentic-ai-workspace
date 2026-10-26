<script lang="ts">
  import { onMount } from 'svelte';
  import { tasks } from '../../logic/workspace-store';
  import type { Grant } from './egress-api';
  import { decideGrant, listGrants, requestGrant } from './egress-api';
  import GrantRow from './GrantRow.svelte';
  import { effectiveStatus } from './egress-time';

  interface Props { projectId: string; initial: Grant[] }
  let { projectId, initial }: Props = $props();
  let grants = $state<Grant[]>([]);
  let busy = $state(false), loading = $state(false);
  let now = $state(Date.now());
  let error = $state(''), success = $state('');
  let host = $state(''), reason = $state(''), taskId = $state('');
  const activeTasks = $derived($tasks.filter((task) => task.projectId === projectId &&
    task.archivedAt === null && task.status !== 'done' && task.status !== 'failed'));
  const pending = $derived(grants.filter((grant) => effectiveStatus(grant, now) === 'pending'));
  const other = $derived(grants.filter((grant) => effectiveStatus(grant, now) !== 'pending'));
  onMount(() => {
    grants = initial;
    void refresh();
    const interval = setInterval(() => { now = Date.now(); }, 1000);
    return () => { clearInterval(interval); };
  });
  $effect(() => {
    if (!activeTasks.some((task) => task.id === taskId)) taskId = activeTasks[0]?.id ?? '';
  });

  async function refresh(): Promise<void> {
    loading = true;
    try {
      grants = await listGrants(projectId);
      error = '';
    } catch {
      error = 'Could not load access requests.';
    } finally {
      loading = false;
    }
  }

  async function submit(): Promise<void> {
    if (taskId === '' || host.trim() === '' || reason.trim() === '') return;
    busy = true; error = ''; success = '';
    try {
      await requestGrant(projectId, taskId, host.trim(), reason.trim());
      host = ''; reason = '';
      success = 'Request submitted. It does not grant internet access until approved.';
      await refresh();
    } catch {
      error = 'Could not request access. Check the hostname, task status, and pending request limit.';
    } finally {
      busy = false;
    }
  }

  function onGrantDecision(grant: Grant, action: 'approve' | 'deny' | 'revoke', hours: number): void {
    void decide(grant, action, hours);
  }

  async function decide(grant: Grant, action: 'approve' | 'deny' | 'revoke', hours: number): Promise<void> {
    busy = true; error = ''; success = '';
    try {
      await decideGrant(projectId, grant.id, action, hours);
      success = action === 'approve' ? 'Access approved for this task.' : 'Request updated.';
      await refresh();
    } catch {
      error = 'Could not apply that decision. The request may have expired or changed.';
      await refresh();
    } finally {
      busy = false;
    }
  }
</script>

<section class="space-y-6" aria-label="Internet access requests">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h2 class="font-semibold text-lg">Internet access</h2>
      <p class="text-sm text-base-content/70">Approve one HTTPS hostname for one task. Access ends automatically.</p>
    </div>
    <button class="btn btn-outline btn-sm" type="button" disabled={loading || busy} onclick={() => void refresh()}>
      {loading ? 'Refreshing…' : 'Refresh'}
    </button>
  </div>
  {#if error !== ''}<p class="alert alert-error text-sm" role="alert">{error}</p>{/if}
  {#if success !== ''}<p class="alert alert-success text-sm" role="status">{success}</p>{/if}
  <form class="rounded-box border border-base-300 p-4 space-y-3" onsubmit={(event) => { event.preventDefault(); void submit(); }}>
    <h3 class="font-medium">Request access manually</h3>
    <p class="text-sm text-base-content/70">For solo tasks or when an agent cannot request access itself.</p>
    <label class="form-control block space-y-1">
      <span class="label text-sm">Task</span>
      <select class="select w-full" bind:value={taskId} required disabled={busy || activeTasks.length === 0}>
        {#each activeTasks as task (task.id)}<option value={task.id}>{task.title}</option>{/each}
      </select>
    </label>
    <label class="form-control block space-y-1">
      <span class="label text-sm">HTTPS hostname</span>
      <input class="input w-full" type="text" autocomplete="off" placeholder="docs.example.com"
        bind:value={host} maxlength="253" required disabled={busy} />
    </label>
    <label class="form-control block space-y-1">
      <span class="label text-sm">Reason</span>
      <textarea class="textarea w-full" rows="2" bind:value={reason} maxlength="500" required disabled={busy}></textarea>
    </label>
    <button type="submit" class="btn btn-primary btn-sm" disabled={busy || activeTasks.length === 0}>Request access</button>
    {#if activeTasks.length === 0}<p class="text-xs text-base-content/70">Start an active task to request access.</p>{/if}
  </form>
  <div class="space-y-3">
    <h3 class="font-medium">Waiting for decision ({pending.length})</h3>
    {#each pending as grant (grant.id)}<GrantRow {grant} {busy} {now} decide={onGrantDecision} />{:else}
      <p class="text-sm text-base-content/70">No pending requests.</p>
    {/each}
  </div>
  <div class="space-y-3">
    <h3 class="font-medium">Previous decisions</h3>
    {#each other as grant (grant.id)}<GrantRow {grant} {busy} {now} decide={onGrantDecision} />{:else}
      <p class="text-sm text-base-content/70">No past decisions.</p>
    {/each}
  </div>
</section>
