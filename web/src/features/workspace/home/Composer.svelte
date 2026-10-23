<script lang="ts">
  import { navigate } from '../../../app/router';
  import ComposerBox from '../components/ComposerBox.svelte';
  import FormField from '../components/FormField.svelte';
  import { homeCopy } from '../content/tasks-copy';
  import { suggestions } from '../content/user';
  import { describeTaskError } from '../logic/task-errors';
  import { attachmentsCanSend, readyPaths, withAttachments, type Attachment } from '../logic/attachments';
  import { createTask } from '../logic/tasks-api';
  import { chosenProject, projects, refresh } from '../logic/workspace-store';

  let text = $state('');
  let project = $state('');
  let mode = $state<'coding' | 'orchestrator'>('coding');
  let attachments = $state<Attachment[]>([]);
  let busy = $state(false);
  let error = $state('');
  let uploadProgress = $state('');

  // Preselect only when there is no real choice to make: the project the person came from, or the only one.
  $effect(() => {
    if (project !== '') return;
    const known = (id: string): boolean => $projects.some((item) => item.id === id);
    if ($chosenProject !== '' && known($chosenProject)) project = $chosenProject;
    else if ($projects.length === 1 && $projects[0] !== undefined) project = $projects[0].id;
  });

  async function send(prompt: string, chosen: string): Promise<void> {
    busy = true;
    error = '';
    if (!attachmentsCanSend(attachments)) {
      busy = false;
      return;
    }
    try {
      const paths = readyPaths(attachments);
      // The model and effort come from Settings: the lead's own, else the person's default.
      const task = await createTask({
        projectId: chosen,
        prompt: withAttachments(prompt, paths),
        team: mode === 'orchestrator',
      });
      text = '';
      attachments = [];
      await refresh();
      navigate(`/app/projects/${task.projectId}/tasks/${task.id}`);
    } catch (failure) {
      error = describeTaskError(failure);
    } finally {
      busy = false;
    }
  }
</script>

{#if $projects.length === 0}
  <p class="rounded-xl border border-dashed border-[var(--brand-border-strong)] p-5 text-[var(--brand-text-secondary)]">
    {homeCopy.noProjects} <a class="link link-hover font-medium text-primary" href="/app/projects">{homeCopy.createProject}</a>
  </p>
{:else}
  <ComposerBox
    id="composer"
    placeholder="Ask a question or tell me what you need…"
    bind:text
    bind:project
    bind:attachments
    projectId={project}
    attachable
    projects={$projects}
    {busy}
    {uploadProgress}
    {error}
    onsend={send}
  >
    {#snippet tools()}
      <div class="min-w-52 flex-1 px-2 py-1">
        <FormField
          id="composer-mode"
          label="Bagaimana task ini dikerjakan?"
          hint="Coding mengerjakan langsung. Orchestrator mengoordinasikan tim."
        >
          {#snippet children({ describedby, invalid })}
            <select
              id="composer-mode"
              class="select select-sm min-h-11 w-full"
              bind:value={mode}
              aria-describedby={describedby}
              aria-invalid={invalid}
              disabled={busy}
            >
              <option value="coding">Coding — langsung</option>
              <option value="orchestrator">Orchestrator — melalui tim</option>
            </select>
          {/snippet}
        </FormField>
      </div>
    {/snippet}
  </ComposerBox>
  <ul class="mt-3 flex flex-wrap gap-2">
    {#each suggestions as suggestion (suggestion)}
      <li><button class="btn btn-sm rounded-full border-[var(--brand-border)] bg-base-100 font-normal" type="button" onclick={() => (text = suggestion)}>{suggestion}</button></li>
    {/each}
  </ul>
{/if}
