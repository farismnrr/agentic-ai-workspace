<script lang="ts">
  import { navigate } from '../../../app/router';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import FormField from '../components/FormField.svelte';
  import SettingsSection from '../settings/parts/SettingsSection.svelte';
  import { projectsCopy } from '../content/projects-copy';
  import { describeProjectError } from '../logic/project-errors';
  import { archiveProject, deleteProject, updateProject } from '../logic/projects-api';
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
  let deleting = $state(false);

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

  async function remove(): Promise<void> {
    deleting = false;
    try {
      await deleteProject(project.id);
      await refresh();
      navigate('/app/projects');
    } catch (error) {
      failed = true;
      message = describeProjectError(error);
    }
  }
</script>

<SettingsSection title="About" description="Basic details about this project.">
  {#snippet action()}
    <a class="link link-hover text-xs font-medium" href="/app/settings/persona">Edit your rules</a>
  {/snippet}
  <form class="flex max-w-xl flex-col gap-4 py-4" onsubmit={save}>
    <FormField id="about-name" label="Name">
      {#snippet children({ describedby, invalid })}<input id="about-name" class="input input-sm w-full" maxlength="80" required bind:value={name} aria-describedby={describedby} aria-invalid={invalid} />{/snippet}
    </FormField>
    <FormField id="about-id" label="Project ID" hint={projectsCopy.idFixed}>
      <p><output id="about-id"><code>{project.slug}</code></output></p>
    </FormField>
    <FormField id="about-description" label="What is it?">
      {#snippet children({ describedby, invalid })}<textarea id="about-description" class="textarea w-full" rows="3" maxlength="500" bind:value={description} aria-describedby={describedby} aria-invalid={invalid}></textarea>{/snippet}
    </FormField>
    <p class="text-sm text-[var(--brand-text-secondary)]">{projectsCopy.rules}</p>
    <div class="flex flex-wrap items-center gap-3">
      <button class="btn btn-primary btn-sm" type="submit">Save changes</button>
      <button class="btn btn-outline btn-error btn-sm" type="button" onclick={() => (archiving = true)}>Archive project</button>
      <button class="btn btn-error btn-sm" type="button" onclick={() => (deleting = true)}>Delete project</button>
      {#if message !== ''}<p class="text-sm {failed ? 'text-error' : 'text-[var(--brand-text-secondary)]'}" role="status">{message}</p>{/if}
    </div>
  </form>
</SettingsSection>
<ConfirmModal
  open={archiving}
  title="Archive {project.name}?"
  body={projectsCopy.archiveBody}
  confirmLabel="Archive project"
  cancelLabel="Keep it"
  oncancel={() => (archiving = false)}
  onconfirm={archive}
/>
<ConfirmModal
  open={deleting}
  title="Delete {project.name} for good?"
  body={projectsCopy.deleteBody}
  confirmLabel="Yes, delete it"
  cancelLabel="Keep it"
  oncancel={() => (deleting = false)}
  onconfirm={remove}
/>
