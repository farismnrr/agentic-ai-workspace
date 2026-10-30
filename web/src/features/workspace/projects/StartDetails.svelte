<script lang="ts">
  import { projectsCopy } from '../content/projects-copy';
  import { newProjectCopy } from '../content/templates-copy';
  import IdField from './IdField.svelte';
  import NameField from './NameField.svelte';
  import RepoPicker from './RepoPicker.svelte';

  interface Props {
    source: 'empty' | 'github' | 'new';
    name?: string | undefined;
    slug?: string | undefined;
    description?: string | undefined;
    repo?: string | undefined;
    repoName?: string | undefined;
    isPrivate?: boolean | undefined;
    busy: boolean;
    error: string;
    onsubmit: () => void;
    oncancel: () => void;
  }

  let {
    source, name = $bindable(''), slug = $bindable(''), description = $bindable(''), repo = $bindable(''), repoName = $bindable(''), isPrivate = $bindable(true),
    busy, error, onsubmit, oncancel,
  }: Props = $props();

  const repoSlug = (text: string): string => text.trim().toLowerCase().replace(/[^a-z0-9._-]+/g, '-').replace(/^-+|-+$/g, '');

  function pick(chosen: string): void {
    repo = chosen;
    if (name.trim() === '') name = chosen.split('/')[1] ?? chosen;
  }

  let idBlocked = $state(false);
  const incomplete = $derived(name.trim() === '' || idBlocked || (source === 'github' && repo === '') || (source === 'new' && repoName.trim() === ''));
</script>

<form class="max-w-xl rounded-xl border border-[var(--brand-border)] bg-base-100 p-4 sm:p-5" onsubmit={(event) => { event.preventDefault(); onsubmit(); }}>
  <h2 class="text-lg font-semibold">{newProjectCopy.detailsHeading}</h2>
  {#if source === 'github'}<RepoPicker value={repo} onpick={pick} />{/if}
  <NameField bind:name />
  <IdField {name} bind:slug onblock={(blocked: boolean) => { idBlocked = blocked; }} />
  <label class="mt-4 block font-medium" for="project-description">What is it? <span class="font-normal text-[var(--brand-text-tertiary)]">(optional)</span></label>
  <textarea id="project-description" class="textarea mt-1 w-full" rows="3" maxlength="500" bind:value={description}></textarea>
  {#if source === 'new'}
    <label class="mt-4 block font-medium" for="project-repo-name">{projectsCopy.github.repoName}</label>
    <input id="project-repo-name" class="input mt-1 w-full" maxlength="100" placeholder={repoSlug(name) || 'my-project'} bind:value={repoName} />
    <label class="mt-3 flex items-center gap-2"><input type="checkbox" class="toggle toggle-sm toggle-primary" bind:checked={isPrivate} /> {projectsCopy.github.private}</label>
  {/if}
  {#if error !== ''}<p class="mt-3 text-sm text-error" role="alert">{error}</p>{/if}
  <div class="mt-6 flex gap-2">
    <button class="btn btn-primary" type="submit" disabled={busy || incomplete}>{newProjectCopy.create[source === 'empty' ? 'project' : source === 'new' ? 'repo' : 'github']}</button>
    <button class="btn btn-ghost" type="button" onclick={oncancel}>{newProjectCopy.cancel}</button>
  </div>
</form>
