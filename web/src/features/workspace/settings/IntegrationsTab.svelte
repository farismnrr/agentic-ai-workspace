<script lang="ts">
  import { onMount } from 'svelte';
  import { githubCopy } from '../content/integrations';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import { disconnectGitHub, loadGitHub, startGitHub, type GitHubStatus } from '../logic/github-api';
  import SettingsSection from './parts/SettingsSection.svelte';

  let status: GitHubStatus | undefined = $state();
  let loading = $state(true);
  let error = $state('');
  let result: 'connected' | 'failed' | undefined = $state();
  let confirming = $state(false);
  let busy = $state(false);

  async function refresh(): Promise<void> {
    try {
      status = await loadGitHub();
      error = '';
    } catch {
      error = githubCopy.failed;
    } finally {
      loading = false;
    }
  }

  async function connect(): Promise<void> {
    busy = true;
    try {
      window.location.assign(await startGitHub());
    } catch {
      error = githubCopy.startFailed;
      busy = false;
    }
  }

  async function disconnect(): Promise<void> {
    confirming = false;
    try {
      await disconnectGitHub();
    } finally {
      await refresh();
    }
  }

  onMount(() => {
    const outcome = new URLSearchParams(window.location.search).get('github');
    if (outcome === 'connected' || outcome === 'failed') {
      result = outcome;
      window.history.replaceState(null, '', window.location.pathname);
    }
    void refresh();
  });
</script>

<SettingsSection title="GitHub" description={githubCopy.intro}>
  {#if result !== undefined}
    <div class="alert my-3 {result === 'connected' ? 'alert-success' : 'alert-error'}" role="status">{githubCopy.result[result]}</div>
  {/if}
  {#if loading}
    <p class="py-3"><span class="loading loading-spinner loading-sm" aria-label="Loading"></span></p>
  {:else if error !== ''}
    <p class="py-3 text-error" role="alert">{error}</p>
  {:else if status !== undefined && !status.configured}
    <p class="py-3 text-sm">{githubCopy.notConfigured}</p>
  {:else if status?.connected === true}
    <div class="flex flex-wrap items-start justify-between gap-3 py-3">
      <div class="min-w-0 flex-1">
        <p class="font-medium">{githubCopy.connected(status.login)} <span class="badge badge-success badge-sm">Connected</span></p>
        <p class="mt-1 text-xs text-[var(--brand-text-secondary)]">{githubCopy.connectedBody}</p>
        {#if status.reposUrl !== ''}
          <p class="mt-2 text-xs">
            <a class="link" href={status.reposUrl} target="_blank" rel="noreferrer noopener">{githubCopy.repos.title}</a>
            · {githubCopy.repos.body}
          </p>
        {/if}
      </div>
      <button class="btn btn-outline btn-error btn-sm" type="button" onclick={() => (confirming = true)}>Disconnect</button>
    </div>
  {:else}
    <div class="flex flex-wrap items-center justify-between gap-3 py-3">
      <div>
        <p class="font-medium">{githubCopy.notConnected.title}</p>
        <p class="mt-1 text-xs text-[var(--brand-text-secondary)]">{githubCopy.notConnected.body}</p>
      </div>
      <button class="btn btn-primary btn-sm" type="button" disabled={busy} onclick={connect}>Connect GitHub</button>
    </div>
  {/if}
</SettingsSection>
<ConfirmModal
  open={confirming}
  title={githubCopy.disconnect.title}
  body={githubCopy.disconnect.body}
  confirmLabel={githubCopy.disconnect.confirm}
  cancelLabel={githubCopy.disconnect.cancel}
  oncancel={() => (confirming = false)}
  onconfirm={disconnect}
/>
