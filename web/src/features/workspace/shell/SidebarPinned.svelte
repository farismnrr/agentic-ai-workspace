<script lang="ts">
  import { loaded, projects, tasks } from '../logic/workspace-store';
  import SkeletonList from '../../../shared/ui/SkeletonList.svelte';
  import SidebarTaskRow from './SidebarTaskRow.svelte';

  const pinned = $derived(
    $tasks.filter((task) => task.pinnedAt !== null).sort((left, right) =>
      Date.parse(right.pinnedAt ?? '') - Date.parse(left.pinnedAt ?? ''),
    ),
  );
  const projectNames = $derived(new Map($projects.map((project) => [project.id, project.name])));
</script>

{#if !$loaded}
  <p class="px-2 pb-1 pt-3 text-xs text-[var(--brand-text-tertiary)]">Pinned</p>
  <SkeletonList variant="sidebar" rows={3} />
{:else if pinned.length > 0}
  <p class="px-2 pb-1 pt-3 text-xs text-[var(--brand-text-tertiary)]">Pinned</p>
  <ul class="flex flex-col" aria-label="Pinned">
    {#each pinned as task (task.id)}
      <SidebarTaskRow {task} projectName={projectNames.get(task.projectId) ?? task.projectId} />
    {/each}
  </ul>
{/if}
