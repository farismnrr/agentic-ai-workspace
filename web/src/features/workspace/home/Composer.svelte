<script lang="ts">
  import { navigate } from '../../../app/router';
  import ComposerBox from '../components/ComposerBox.svelte';
  import { homeCopy } from '../content/tasks-copy';
  import { suggestions } from '../content/user';
  import { describeTaskError } from '../logic/task-errors';
  import { createTask } from '../logic/tasks-api';
  import type { ApprovalMode } from '../logic/task-types';
  import { chosenProject, projects, refresh } from '../logic/workspace-store';

  let text = $state('');
  let project = $state('');
  let busy = $state(false);
  let error = $state('');

  const fallback = $derived($chosenProject !== '' ? $chosenProject : ($projects[0]?.id ?? ''));
  const selected = $derived($projects.some((item) => item.id === project) ? project : fallback);

  async function send(prompt: string, mode: ApprovalMode, chosen: string): Promise<void> {
    busy = true;
    error = '';
    try {
      const task = await createTask({ projectId: chosen || selected, prompt, mode });
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
  <ComposerBox id="composer" placeholder="Describe what you need in plain words…" bind:text bind:project projects={$projects} {busy} {error} onsend={send} />
  <ul class="mt-3 flex flex-wrap gap-2">
    {#each suggestions as suggestion (suggestion)}
      <li><button class="btn btn-sm rounded-full border-[var(--brand-border)] bg-base-100 font-normal" type="button" onclick={() => (text = suggestion)}>{suggestion}</button></li>
    {/each}
  </ul>
{/if}
