<script lang="ts">
  import { describeProjectError } from '../logic/project-errors';
  import { projectsCopy } from '../content/projects-copy';
  import { loadGitHub } from '../logic/github-api';
  import { createNewProject } from '../logic/create-project';
  import RepoPicker from './RepoPicker.svelte';
  import TemplatePicker from './TemplatePicker.svelte';
  import type { Project } from '../logic/task-types';

  interface Props {
    show?: boolean | undefined;
    oncreated: (project: Project) => void;
  }

  let { show = $bindable(false), oncreated }: Props = $props();
  let dialog = $state<HTMLDialogElement>();
  let name = $state('');
  let description = $state('');
  let busy = $state(false);
  let error = $state('');
  /** off: the server has no GitHub App; connect: it has one but the person has not connected; ready: connected. */
  let github = $state<'off' | 'connect' | 'ready'>('off');
  let source = $state<'empty' | 'github' | 'new'>('empty');
  let template = $state('');
  let repo = $state('');
  let repoName = $state('');
  let isPrivate = $state(true);

  $effect(() => {
    if (show && dialog?.open !== true) dialog?.showModal();
    if (!show && dialog?.open === true) dialog.close();
  });

  $effect(() => {
    if (show) void loadGitHub().then((status) => { github = status.connected ? 'ready' : status.configured ? 'connect' : 'off'; }, () => { github = 'off'; });
  });

  function pick(chosen: string): void {
    repo = chosen;
    if (name.trim() === '') name = chosen.split('/')[1] ?? chosen;
  }

  const slug = (text: string): string => text.trim().toLowerCase().replace(/[^a-z0-9._-]+/g, '-').replace(/^-+|-+$/g, '');

  async function submit(event: SubmitEvent): Promise<void> {
    event.preventDefault();
    busy = true;
    error = '';
    try {
      const project = await createNewProject({ name, description, source, template, repo, repoName, isPrivate });
      name = '';
      repo = '';
      repoName = '';
      source = 'empty';
      template = '';
      description = '';
      show = false;
      oncreated(project);
    } catch (failure) {
      error = describeProjectError(failure);
    } finally {
      busy = false;
    }
  }
</script>

<dialog bind:this={dialog} class="modal" onclose={() => (show = false)} aria-labelledby="new-project-title">
  <form class="modal-box max-w-md" onsubmit={submit}>
    <h2 id="new-project-title" class="text-base font-semibold">New project</h2>
    {#if source !== 'github'}<TemplatePicker bind:value={template} />{/if}
    {#if github !== 'off'}
      <div class="join mt-4" role="group" aria-label="Where it starts">
        <button type="button" class="btn btn-sm join-item {source === 'empty' ? 'btn-primary' : ''}" aria-pressed={source === 'empty'} onclick={() => { source = 'empty'; }}>{projectsCopy.github.empty}</button>
        <button type="button" class="btn btn-sm join-item {source === 'github' ? 'btn-primary' : ''} {github === 'connect' ? '!border-[var(--brand-border-strong)] !bg-base-100 !text-[var(--brand-text-secondary)] cursor-not-allowed' : ''}" aria-pressed={source === 'github'} aria-disabled={github === 'connect'} aria-describedby={github === 'connect' ? 'github-hint' : undefined} onclick={() => { if (github === 'ready') { source = 'github'; template = ''; } }}>{projectsCopy.github.fromGitHub}</button>
        <button type="button" class="btn btn-sm join-item {source === 'new' ? 'btn-primary' : ''} {github === 'connect' ? '!border-[var(--brand-border-strong)] !bg-base-100 !text-[var(--brand-text-secondary)] cursor-not-allowed' : ''}" aria-pressed={source === 'new'} aria-disabled={github === 'connect'} aria-describedby={github === 'connect' ? 'github-hint' : undefined} onclick={() => { if (github === 'ready') { source = 'new'; if (repoName === '') repoName = slug(name); } }}>{projectsCopy.github.newRepo}</button>
      </div>
      {#if github === 'connect'}
        <p id="github-hint" class="mt-2 text-sm text-[var(--brand-text-secondary)]">{projectsCopy.github.connectHint} <a class="link link-primary font-medium" href="/app/settings/integrations">{projectsCopy.github.connectLink}</a></p>
      {/if}
      {#if source === 'github'}<RepoPicker value={repo} onpick={pick} />{/if}
      {#if source === 'new'}
        <label class="mt-4 block font-medium" for="project-repo-name">{projectsCopy.github.repoName}</label>
        <input id="project-repo-name" class="input input-sm mt-1 w-full" maxlength="100" placeholder={slug(name) || 'my-project'} bind:value={repoName} />
        <label class="mt-3 flex items-center gap-2"><input type="checkbox" class="toggle toggle-sm toggle-primary" bind:checked={isPrivate} /> {projectsCopy.github.private}</label>
      {/if}
    {/if}
    <label class="mt-4 block font-medium" for="project-name">Name</label>
    <input id="project-name" class="input input-sm mt-1 w-full" maxlength="80" placeholder="Bakery website" required bind:value={name} />
    <label class="mt-4 block font-medium" for="project-description">What is it? <span class="font-normal text-[var(--brand-text-tertiary)]">(optional)</span></label>
    <textarea id="project-description" class="textarea mt-1 w-full" rows="3" maxlength="500" bind:value={description}></textarea>
    {#if error !== ''}<p class="mt-3 text-sm text-error" role="alert">{error}</p>{/if}
    <div class="modal-action">
      <button class="btn btn-ghost btn-sm" type="button" onclick={() => (show = false)}>Cancel</button>
      <button class="btn btn-primary btn-sm" type="submit" disabled={busy || name.trim() === '' || (source === 'github' && repo === '') || (source === 'new' && repoName.trim() === '')}>{source === 'empty' ? 'Create project' : source === 'new' ? 'Create repository and project' : 'Create and import'}</button>
    </div>
  </form>
  <button class="modal-backdrop" type="button" aria-label="Close" onclick={() => (show = false)}></button>
</dialog>
