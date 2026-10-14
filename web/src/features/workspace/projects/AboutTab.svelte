<script lang="ts">
  import { navigate } from '../../../app/router';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import { projectsCopy } from '../content/projects-copy';
  import { describeProjectError } from '../logic/project-errors';
  import { archiveProject, updateProject } from '../logic/projects-api';
  import type { Project } from '../logic/task-types';
  import { refresh } from '../logic/workspace-store';

  interface Props {
    project: Project;
  }

  const { project }: Props = $props();
  let name = $state('');
  let description = $state('');
  let loadedFor = '';
  let message = $state('');
  let failed = $state(false);
  let archiving = $state(false);

  $effect(() => {
    if (loadedFor !== project.id) {
      loadedFor = project.id;
      name = project.name;
      description = project.description;
    }
  });

  async function save(event: SubmitEvent): Promise<void> {
    event.preventDefault();
    try {
      await updateProject(project.id, name, description);
      await refresh();
      failed = false;
      message = 'Saved.';
    } catch (error) {
      failed = true;
      message = describeProjectError(error);
    }
  }

  async function archive(): Promise<void> {
    archiving = false;
    try {
      await archiveProject(project.id);
      await refresh();
      navigate('/app/projects');
    } catch (error) {
      failed = true;
      message = describeProjectError(error);
    }
  }
</script>

<form class="card flex max-w-xl flex-col gap-4 border border-[var(--brand-border)] bg-base-100 p-5" onsubmit={save}>
  <div>
    <label class="font-medium" for="about-name">Name</label>
    <input id="about-name" class="input input-sm mt-1 w-full" maxlength="80" required bind:value={name} />
  </div>
  <div>
    <label class="font-medium" for="about-description">What is it?</label>
    <textarea id="about-description" class="textarea mt-1 w-full" rows="3" maxlength="500" bind:value={description}></textarea>
  </div>
  <p class="text-sm text-[var(--brand-text-secondary)]">{projectsCopy.rules}</p>
  <div class="flex items-center gap-3">
    <button class="btn btn-primary btn-sm" type="submit">Save changes</button>
    <button class="btn btn-outline btn-error btn-sm" type="button" onclick={() => (archiving = true)}>Archive project</button>
    {#if message !== ''}<p class="text-sm {failed ? 'text-error' : 'text-[var(--brand-text-secondary)]'}" role="status">{message}</p>{/if}
  </div>
</form>
<a class="link link-hover mt-4 inline-block font-medium text-primary" href="/app/settings/persona">Edit your rules</a>
<ConfirmModal
  open={archiving}
  title="Archive {project.name}?"
  body={projectsCopy.archiveBody}
  confirmLabel="Archive project"
  cancelLabel="Keep it"
  oncancel={() => (archiving = false)}
  onconfirm={archive}
/>
