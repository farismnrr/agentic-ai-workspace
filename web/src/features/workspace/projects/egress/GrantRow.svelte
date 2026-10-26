<script lang="ts">
  import type { Grant } from './egress-api';

  interface Props {
    grant: Grant;
    busy: boolean;
    decide: (grant: Grant, action: 'approve' | 'deny' | 'revoke', hours: number) => void;
  }

  let { grant, busy, decide }: Props = $props();
  let hours = $state(24);
  const pending = $derived(grant.status === 'pending');
  const active = $derived(grant.status === 'approved');
  const color = $derived(active ? 'badge-success' : pending ? 'badge-warning' : 'badge-ghost');
  const until = $derived(new Date(grant.expiresAt).toLocaleString());
</script>

<article class="rounded-box border border-base-300 bg-base-100 p-4 space-y-3">
  <div class="flex flex-wrap items-start justify-between gap-2">
    <div class="min-w-0">
      <p class="break-all font-medium">{grant.host}</p>
      <a class="link link-primary text-xs" href="/app/projects/{encodeURIComponent(grant.projectId)}/tasks/{encodeURIComponent(grant.taskId)}">
        Open task
      </a>
    </div>
    <span class="badge {color}">{grant.status}</span>
  </div>
  <p class="text-sm text-base-content/80 whitespace-pre-wrap break-words">{grant.reason}</p>
  <dl class="grid grid-cols-1 gap-1 text-xs text-base-content/70 sm:grid-cols-2">
    <div><dt class="inline font-medium">Requester: </dt><dd class="inline break-all">{grant.requester} ({grant.requestedByKind})</dd></div>
    <div><dt class="inline font-medium">{pending ? 'Request expires: ' : 'Access ends: '}</dt><dd class="inline">{until}</dd></div>
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
