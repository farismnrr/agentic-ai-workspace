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
  let query = $state('');
  let active = $state(0);

  const shown = $derived(repos.filter((repo) => repo.name.toLowerCase().includes(query.trim().toLowerCase())));

  $effect(() => {
    void loadRepos().then(
      (list) => { repos = list; status = 'ready'; },
      () => { status = 'failed'; },
    );
  });

  function move(event: KeyboardEvent): void {
    const current = shown[active];
    if (event.key === 'ArrowDown') active = Math.min(active + 1, shown.length - 1);
    else if (event.key === 'ArrowUp') active = Math.max(active - 1, 0);
    else if (event.key === 'Enter' && current !== undefined) onpick(current.name);
    else return;
    event.preventDefault();
  }
</script>

<label class="mt-4 block font-medium" for="project-repo-search">{projectsCopy.github.pick}</label>
{#if status === 'loading'}
  <p class="mt-1 text-sm text-[var(--brand-text-secondary)]">{projectsCopy.github.loading}</p>
{:else if status === 'failed'}
  <p class="mt-1 text-sm text-error" role="alert">{projectsCopy.github.failed}</p>
{:else if repos.length === 0}
  <p class="mt-1 text-sm text-[var(--brand-text-secondary)]">{projectsCopy.github.none}</p>
{:else}
  <input
    id="project-repo-search"
    class="input input-sm mt-1 w-full"
    type="search"
    placeholder={projectsCopy.github.search}
    autocomplete="off"
    role="combobox"
    aria-expanded="true"
    aria-controls="project-repo-list"
    bind:value={query}
    oninput={() => { active = 0; }}
    onkeydown={move}
  />
  <ul id="project-repo-list" role="listbox" class="menu mt-1 block max-h-40 w-full overflow-y-auto rounded-lg border border-[var(--brand-border)] bg-base-100 p-1">
    {#each shown as repo, index (repo.name)}
      <li role="option" aria-selected={repo.name === value}>
        <button type="button" class="flex w-full justify-between gap-2 {repo.name === value ? 'bg-primary/15 font-medium text-base-content' : ''} {index === active && repo.name !== value ? 'bg-base-200' : ''}" onclick={() => { onpick(repo.name); }}>
          <span class="truncate">{repo.name}</span>
          {#if repo.private}<span class="badge badge-ghost badge-sm shrink-0">private</span>{/if}
        </button>
      </li>
    {:else}
      <li class="px-3 py-2 text-sm text-[var(--brand-text-secondary)]">{projectsCopy.github.noMatch}</li>
    {/each}
  </ul>
{/if}
