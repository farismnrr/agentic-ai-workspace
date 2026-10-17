<script lang="ts">
  import { describeProjectError } from '../logic/project-errors';
  import { projectsCopy } from '../content/projects-copy';
  import { loadGitHub } from '../logic/github-api';
  import { archiveProject, createProject, importRepo } from '../logic/projects-api';
  import RepoPicker from './RepoPicker.svelte';
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
  let canImport = $state(false);
  let source = $state<'empty' | 'github'>('empty');
  let repo = $state('');

  $effect(() => {
    if (show && dialog?.open !== true) dialog?.showModal();
    if (!show && dialog?.open === true) dialog.close();
  });

  $effect(() => {
    if (show) void loadGitHub().then((status) => { canImport = status.connected; }, () => { canImport = false; });
  });

  function pick(chosen: string): void {
    repo = chosen;
    if (name.trim() === '') name = chosen.split('/')[1] ?? chosen;
  }

  async function create(): Promise<Project> {
    const project = await createProject(name, description);
    if (source !== 'github') return project;
    try {
      await importRepo(project.id, repo);
    } catch (failure) {
      await archiveProject(project.id).catch(() => undefined);
      throw failure;
    }
    return project;
  }

  async function submit(event: SubmitEvent): Promise<void> {
    event.preventDefault();
    busy = true;
    error = '';
    try {
      const project = await create();
      name = '';
      repo = '';
      source = 'empty';
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
    {#if canImport}
      <div class="join mt-4" role="group" aria-label="Where it starts">
        <button type="button" class="btn btn-sm join-item {source === 'empty' ? 'btn-primary' : ''}" aria-pressed={source === 'empty'} onclick={() => { source = 'empty'; }}>{projectsCopy.github.empty}</button>
        <button type="button" class="btn btn-sm join-item {source === 'github' ? 'btn-primary' : ''}" aria-pressed={source === 'github'} onclick={() => { source = 'github'; }}>{projectsCopy.github.fromGitHub}</button>
      </div>
      {#if source === 'github'}<RepoPicker value={repo} onpick={pick} />{/if}
    {/if}
    <label class="mt-4 block font-medium" for="project-name">Name</label>
    <input id="project-name" class="input input-sm mt-1 w-full" maxlength="80" placeholder="Bakery website" required bind:value={name} />
    <label class="mt-4 block font-medium" for="project-description">What is it? <span class="font-normal text-[var(--brand-text-tertiary)]">(optional)</span></label>
    <textarea id="project-description" class="textarea mt-1 w-full" rows="3" maxlength="500" bind:value={description}></textarea>
    {#if error !== ''}<p class="mt-3 text-sm text-error" role="alert">{error}</p>{/if}
    <div class="modal-action">
      <button class="btn btn-ghost btn-sm" type="button" onclick={() => (show = false)}>Cancel</button>
      <button class="btn btn-primary btn-sm" type="submit" disabled={busy || name.trim() === '' || (source === 'github' && repo === '')}>{source === 'github' ? 'Create and import' : 'Create project'}</button>
    </div>
  </form>
  <button class="modal-backdrop" type="button" aria-label="Close" onclick={() => (show = false)}></button>
</dialog>
