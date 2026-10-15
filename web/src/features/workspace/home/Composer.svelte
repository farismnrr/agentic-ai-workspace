<script lang="ts">
  import { navigate } from '../../../app/router';
  import ComposerBox from '../components/ComposerBox.svelte';
  import ModelPicker from './ModelPicker.svelte';
  import { homeCopy } from '../content/tasks-copy';
  import { suggestions } from '../content/user';
  import { describeTaskError } from '../logic/task-errors';
  import { createTask } from '../logic/tasks-api';
  import { chosenProject, projects, refresh } from '../logic/workspace-store';

  let text = $state('');
  let project = $state('');
  let model = $state('');
  let effort = $state('');
  let busy = $state(false);
  let error = $state('');

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
    try {
      const [providerId = '', name = ''] = model.split('::');
      const task = await createTask({ projectId: chosen, prompt, effort, ...(model === '' ? {} : { providerId, model: name }) });
      text = '';
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
    projects={$projects}
    {busy}
    {error}
    onsend={send}
  >
    {#snippet tools()}<ModelPicker bind:model bind:effort disabled={busy} />{/snippet}
  </ComposerBox>
  <ul class="mt-3 flex flex-wrap gap-2">
    {#each suggestions as suggestion (suggestion)}
      <li><button class="btn btn-sm rounded-full border-[var(--brand-border)] bg-base-100 font-normal" type="button" onclick={() => (text = suggestion)}>{suggestion}</button></li>
    {/each}
  </ul>
{/if}
