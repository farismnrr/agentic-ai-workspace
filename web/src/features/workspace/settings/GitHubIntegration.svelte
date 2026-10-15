<script lang="ts">
  import { onMount } from 'svelte';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import { githubCopy } from '../content/integrations';
  import { disconnectGitHub, loadGitHub, startGitHub, type GitHubStatus } from '../logic/github-api';
  import IntegrationRow from './parts/IntegrationRow.svelte';

  let status: GitHubStatus | undefined = $state();
  let loading = $state(true);
  let problem = $state('');
  let confirming = $state(false);
  let busy = $state(false);

  const description = $derived.by(() => {
    if (problem !== '') return problem;
    if (status === undefined) return githubCopy.description;
    if (!status.configured) return githubCopy.notConfigured;
    return status.connected ? githubCopy.connected(status.login) : githubCopy.description;
  });

  async function refresh(): Promise<void> {
    try {
      status = await loadGitHub();
    } catch {
      problem = githubCopy.loadFailed;
    } finally {
      loading = false;
    }
  }

  async function connect(): Promise<void> {
    busy = true;
    try {
      window.location.assign(await startGitHub());
    } catch {
      problem = githubCopy.startFailed;
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
    if (outcome === 'failed') problem = githubCopy.failed;
    if (outcome !== null) window.history.replaceState(null, '', window.location.pathname);
    void refresh();
  });
</script>

<IntegrationRow logo="github" title={githubCopy.name} {description} problem={problem !== ''}>
  {#if loading}
    <span class="loading loading-spinner loading-sm" aria-label="Loading"></span>
  {:else if status?.connected === true}
    {#if status.reposUrl !== ''}
      <a class="btn btn-outline btn-sm" href={status.reposUrl} target="_blank" rel="noreferrer noopener">{githubCopy.repos.button}</a>
    {/if}
    <button class="btn btn-outline btn-error btn-sm" type="button" onclick={() => (confirming = true)}>Disconnect</button>
  {:else if status?.configured === true}
    <button class="btn btn-primary btn-sm" type="button" disabled={busy} onclick={connect}>Connect GitHub</button>
  {/if}
</IntegrationRow>
<ConfirmModal
  open={confirming}
  title={githubCopy.disconnect.title}
  body={githubCopy.disconnect.body}
  confirmLabel={githubCopy.disconnect.confirm}
  cancelLabel={githubCopy.disconnect.cancel}
  oncancel={() => (confirming = false)}
  onconfirm={disconnect}
/>
