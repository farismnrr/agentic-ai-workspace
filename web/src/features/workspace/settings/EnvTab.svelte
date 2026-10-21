<script lang="ts">
  import EmptyState from '../../../shared/ui/EmptyState.svelte';
  import { chosenProject, projects } from '../logic/workspace-store';
  import DatabaseSection from './env/DatabaseSection.svelte';
  import VariablesSection from './env/VariablesSection.svelte';

  let projectId = $state('');
  let variableRevision = $state(0);

  function refreshVariables(): void {
    variableRevision += 1;
  }

  $effect(() => {
    if (!$projects.some((project) => project.id === projectId)) {
      const preferred = $projects.find((project) => project.id === $chosenProject);
      projectId = preferred?.id ?? $projects[0]?.id ?? '';
    }
  });
</script>

<div class="space-y-8">
  <div>
    <h2 class="text-base font-semibold">Project environment</h2>
    <p class="mt-1 text-sm text-[var(--brand-text-secondary)]">Values are available only to runs in the selected project.</p>
    {#if $projects.length > 0}
      <label class="mt-4 block max-w-md">
        <span class="label-text mb-1 block">Project</span>
        <select class="select select-sm w-full" bind:value={projectId}>
          {#each $projects as project (project.id)}<option value={project.id}>{project.name}</option>{/each}
        </select>
      </label>
    {/if}
  </div>
  {#if projectId === ''}
    <EmptyState icon="folder" title="No projects yet" body="Create a project first to add environment variables." bordered />
  {:else}
    {#key projectId}
      <VariablesSection {projectId} revision={variableRevision} />
      <DatabaseSection {projectId} onenvchange={refreshVariables} />
    {/key}
  {/if}
</div>
