<script lang="ts">
  import { onMount } from 'svelte';
  import { request } from '../../../shared/api/http';

  interface PairStatus {
    connected: boolean;
    available: boolean;
    url?: string;
    expiresAt?: string;
  }
  interface Props { taskId: string }

  const { taskId }: Props = $props();
  const path = $derived(`/api/tasks/${encodeURIComponent(taskId)}/molab`);
  let pairing = $state<PairStatus>({ connected: false, available: false });
  let expanded = $state(false);
  let url = $state('');
  let token = $state('');
  let busy = $state(false);
  let error = $state('');

  onMount(() => {
    let active = true;
    const refresh = (): void => {
      if (busy) return;
      void request<PairStatus>('GET', path).then((status) => {
        if (active && !busy) pairing = status;
      }).catch(() => {
        if (active) error = 'Could not check the MoLab pairing.';
      });
    };
    refresh();
    const poll = window.setInterval(refresh, 30_000);
    return () => {
      active = false;
      window.clearInterval(poll);
      token = '';
    };
  });

  async function connect(event: SubmitEvent): Promise<void> {
    event.preventDefault();
    if (busy || !url.trim() || !token) return;
    busy = true;
    error = '';
    try {
      pairing = await request<PairStatus>('POST', path, { url: url.trim(), token });
      expanded = false;
      url = '';
      token = '';
    } catch {
      error = 'Could not pair. Check the temporary MoLab URL and token.';
    } finally {
      busy = false;
    }
  }

  async function disconnect(): Promise<void> {
    if (busy) return;
    busy = true;
    error = '';
    try {
      await request<undefined>('DELETE', path);
      pairing = { connected: false, available: true };
      token = '';
      url = '';
      expanded = false;
    } catch {
      error = 'Could not disconnect. Please retry.';
    } finally {
      busy = false;
    }
  }
</script>

<section class="border-b border-[var(--brand-border)] bg-base-100 px-4 py-2" aria-label="MoLab pairing">
  <div class="flex flex-wrap items-center gap-2 text-sm">
    <span class="font-medium">MoLab notebook</span>
    <span class="text-[var(--brand-text-secondary)]">{pairing.connected ? 'Pairing saved for this task (not yet verified)' : 'Not paired'}</span>
    {#if pairing.connected}
      <button type="button" class="btn btn-ghost btn-xs" onclick={disconnect} disabled={busy}>Disconnect</button>
    {:else if pairing.available}
      <button type="button" class="btn btn-ghost btn-xs" onclick={() => (expanded = !expanded)} aria-expanded={expanded}>
        {expanded ? 'Close' : 'Connect'}
      </button>
    {/if}
  </div>
  {#if !pairing.available}
    <p class="mt-1 text-xs text-[var(--brand-text-secondary)]">
      Task-scoped network access must be enabled by an administrator before pairing.
    </p>
  {/if}
  {#if pairing.connected}
    <p class="mt-1 break-all text-xs text-[var(--brand-text-secondary)]">
      {pairing.url} · Expires {pairing.expiresAt ? new Date(pairing.expiresAt).toLocaleTimeString() : 'soon'}.
      Agent uses marimo_execute. Token is never shown or saved to the project.
    </p>
  {:else if expanded}
    <form onsubmit={connect} class="mt-2 flex max-w-xl flex-col gap-2">
      <label class="text-xs font-medium" for="molab-url">Temporary notebook URL</label>
      <input id="molab-url" class="input input-bordered input-sm w-full" type="url" bind:value={url}
        placeholder="https://sb-xxxx.sb.molab.run/" autocomplete="off" spellcheck="false" required />
      <label class="text-xs font-medium" for="molab-token">MoLab token (private)</label>
      <input id="molab-token" class="input input-bordered input-sm w-full" type="password" bind:value={token}
        autocomplete="off" spellcheck="false" required />
      <p class="text-xs text-[var(--brand-text-secondary)]">
        Stored only in server memory for this task (up to 30 minutes). Not added to the chat, logs, or Project Env.
      </p>
      <button type="submit" class="btn btn-primary btn-sm self-start" disabled={busy || !url.trim() || !token}>
        {busy ? 'Connecting…' : 'Save temporary pairing'}
      </button>
    </form>
  {/if}
  {#if error}<p class="mt-1 text-xs text-error" role="alert">{error}</p>{/if}
</section>
