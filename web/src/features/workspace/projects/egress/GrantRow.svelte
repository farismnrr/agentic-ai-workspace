<script lang="ts">
  import type { Grant } from './egress-api';
  import { effectiveStatus, remainingLabel } from './egress-time';

  interface Props {
    grant: Grant;
    busy: boolean;
    now: number;
    decide: (grant: Grant, action: 'approve' | 'deny' | 'revoke', hours: number) => void;
  }

  let { grant, busy, now, decide }: Props = $props();
  let hours = $state(24);
  const status = $derived(effectiveStatus(grant, now));
  const pending = $derived(status === 'pending');
  const active = $derived(status === 'approved');
  const remaining = $derived(remainingLabel(grant.expiresAt, now));
  const color = $derived(active ? 'badge-success' : pending ? 'badge-warning' : 'badge-ghost');
  const until = $derived(new Date(grant.expiresAt).toLocaleString());
  const created = $derived(new Date(grant.createdAt).toLocaleString());
</script>

<article class="space-y-3 rounded-xl border border-[var(--brand-border)] bg-base-100 p-4">
  <div class="flex flex-wrap items-start justify-between gap-2">
    <div class="min-w-0">
      <p class="break-all font-medium">{grant.host}</p>
      <a class="link link-primary text-xs" href="/app/projects/{encodeURIComponent(grant.projectId)}/tasks/{encodeURIComponent(grant.taskId)}">
        Open task
      </a>
    </div>
    <span class="badge {color}">{status}</span>
  </div>
  <p class="text-sm text-base-content/80 whitespace-pre-wrap break-words">{grant.reason}</p>
  <dl class="grid grid-cols-1 gap-1 text-xs text-base-content/70 sm:grid-cols-2">
    <div><dt class="inline font-medium">Requester: </dt><dd class="inline break-all">{grant.requester} ({grant.requestedByKind})</dd></div>
    <div><dt class="inline font-medium">Created: </dt><dd class="inline">{created}</dd></div>
    <div><dt class="inline font-medium">{status === 'expired' ? 'Expired at: ' : pending ? 'Request expires: ' : 'Access ends: '}</dt><dd class="inline">{until}</dd></div>
    {#if pending || active}<div><dt class="inline font-medium">Time left: </dt><dd class="inline">{remaining}</dd></div>{/if}
    {#if grant.decidedBy}<div><dt class="inline font-medium">Decided by: </dt><dd class="inline break-all">{grant.decidedBy}</dd></div>{/if}
  </dl>
  {#if pending || active}
    <div class="flex flex-wrap items-center gap-2">
      {#if pending}
        <label class="text-xs" for="grant-ttl-{grant.id}">Lifetime</label>
        <select id="grant-ttl-{grant.id}" class="select select-sm" bind:value={hours} disabled={busy}>
          <option value={1}>1 hour</option>
          <option value={24}>24 hours</option>
          <option value={72}>3 days</option>
          <option value={168}>7 days</option>
        </select>
        <button class="btn btn-primary btn-sm" type="button" disabled={busy}
          onclick={() => { decide(grant, 'approve', hours); }}>Approve</button>
        <button class="btn btn-outline btn-sm" type="button" disabled={busy}
          onclick={() => { decide(grant, 'deny', hours); }}>Deny</button>
      {:else}
        <button class="btn btn-error btn-outline btn-sm" type="button" disabled={busy}
          onclick={() => { decide(grant, 'revoke', hours); }}>Revoke access</button>
      {/if}
    </div>
  {/if}
</article>
