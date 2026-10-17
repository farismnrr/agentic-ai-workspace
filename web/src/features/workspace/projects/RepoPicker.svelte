<script lang="ts">
  import { projectsCopy } from '../content/projects-copy';
  import { loadRepos, type GitHubRepo } from '../logic/github-api';

  interface Props {
    /** The chosen repository as owner/name; empty until one is picked. */
    value: string;
    onpick: (repo: string) => void;
  }

  const { value, onpick }: Props = $props();
  let repos = $state<GitHubRepo[]>([]);
  let status = $state<'loading' | 'ready' | 'failed'>('loading');

  $effect(() => {
    void loadRepos().then(
      (list) => { repos = list; status = 'ready'; },
      () => { status = 'failed'; },
    );
  });
</script>

<label class="mt-4 block font-medium" for="project-repo">{projectsCopy.github.pick}</label>
{#if status === 'loading'}
  <p class="mt-1 text-sm text-[var(--brand-text-secondary)]">{projectsCopy.github.loading}</p>
{:else if status === 'failed'}
  <p class="mt-1 text-sm text-error" role="alert">{projectsCopy.github.failed}</p>
{:else if repos.length === 0}
  <p class="mt-1 text-sm text-[var(--brand-text-secondary)]">{projectsCopy.github.none}</p>
{:else}
  <select id="project-repo" class="select select-sm mt-1 w-full" {value} onchange={(event) => { onpick(event.currentTarget.value); }}>
    <option value="" disabled selected={value === ''}>—</option>
    {#each repos as repo (repo.name)}
      <option value={repo.name}>{repo.name}{repo.private ? ' (private)' : ''}</option>
    {/each}
  </select>
{/if}
