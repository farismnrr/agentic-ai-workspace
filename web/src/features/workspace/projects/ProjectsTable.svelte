<script lang="ts">
  import ProjectTile from '../components/ProjectTile.svelte';
  import { countByStatus, tasksOf } from '../logic/derive';
  import type { Project } from '../logic/task-types';
  import { timeAgo } from '../logic/time';
  import { tasks } from '../logic/workspace-store';

  interface Props {
    items: Project[];
  }

  const { items }: Props = $props();
</script>

<div class="overflow-x-auto">
  <table class="table">
    <thead>
      <tr><th>Project</th><th class="hidden sm:table-cell">Needs you</th><th class="hidden md:table-cell">Working</th><th class="hidden md:table-cell">Done</th><th class="text-right">Created</th></tr>
    </thead>
    <tbody>
      {#each items as project (project.id)}
        {@const counts = countByStatus(tasksOf($tasks, project.id))}
        {@const waiting = counts['needs-input'] + counts.review + counts.failed}
        <tr class="hover:bg-base-200/60">
          <td>
            <a class="flex items-center gap-3" href="/app/projects/{project.id}">
              <ProjectTile name={project.name} id={project.id} class="size-9 text-base" />
              <span class="flex flex-col"><span class="flex items-center gap-2"><span class="font-medium">{project.name}</span>{#if project.source !== ''}<span class="badge badge-soft badge-sm">Connected app</span>{/if}</span>{#if project.description !== ''}<span class="max-w-md truncate text-sm text-[var(--brand-text-secondary)]">{project.description}</span>{/if}</span>
            </a>
          </td>
          <td class="hidden sm:table-cell">{#if waiting > 0}<span class="badge badge-primary badge-soft">{waiting} waiting</span>{:else}<span class="text-[var(--brand-text-tertiary)]">None</span>{/if}</td>
          <td class="hidden md:table-cell">{counts.working + counts.planning + counts.queued}</td>
          <td class="hidden md:table-cell">{counts.done}</td>
          <td class="text-right text-[var(--brand-text-secondary)]">{timeAgo(project.createdAt)}</td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>
