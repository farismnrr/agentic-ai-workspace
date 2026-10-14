<script lang="ts">
  import { describeProjectError } from '../logic/project-errors';
  import { createProject } from '../logic/projects-api';
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

  $effect(() => {
    if (show && dialog?.open !== true) dialog?.showModal();
    if (!show && dialog?.open === true) dialog.close();
  });

  async function submit(event: SubmitEvent): Promise<void> {
    event.preventDefault();
    busy = true;
    error = '';
    try {
      const project = await createProject(name, description);
      name = '';
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
    <label class="mt-4 block font-medium" for="project-name">Name</label>
    <input id="project-name" class="input input-sm mt-1 w-full" maxlength="80" placeholder="Bakery website" required bind:value={name} />
    <label class="mt-4 block font-medium" for="project-description">What is it? <span class="font-normal text-[var(--brand-text-tertiary)]">(optional)</span></label>
    <textarea id="project-description" class="textarea mt-1 w-full" rows="3" maxlength="500" bind:value={description}></textarea>
    {#if error !== ''}<p class="mt-3 text-sm text-error" role="alert">{error}</p>{/if}
    <div class="modal-action">
      <button class="btn btn-ghost btn-sm" type="button" onclick={() => (show = false)}>Cancel</button>
      <button class="btn btn-primary btn-sm" type="submit" disabled={busy || name.trim() === ''}>Create project</button>
    </div>
  </form>
  <button class="modal-backdrop" type="button" aria-label="Close" onclick={() => (show = false)}></button>
</dialog>
