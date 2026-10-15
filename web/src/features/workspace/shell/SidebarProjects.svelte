<script lang="ts">
  import { route } from '../../../app/router';
  import ProjectTile from '../components/ProjectTile.svelte';
  import { needsYouCount } from '../logic/derive';
  import { projects, tasks } from '../logic/workspace-store';

  const active = $derived('projectId' in $route ? $route.projectId : undefined);
</script>

<details class="group mt-3" open>
  <summary class="flex cursor-pointer list-none items-center gap-1 px-2 pb-1 text-xs text-[var(--brand-text-tertiary)]">
    Projects <span class="transition-transform group-open:rotate-90" aria-hidden="true">›</span>
  </summary>
  <ul class="flex flex-col">
    {#each $projects.slice(0, 8) as project (project.id)}
      {@const waiting = needsYouCount($tasks, project.id)}
      <li>
        <a
          href="/app/projects/{project.id}"
          aria-current={active === project.id ? 'page' : undefined}
          class="flex items-center gap-2 rounded-lg px-2 py-1.5 hover:bg-base-100/70 {active === project.id ? 'bg-base-300/60 font-medium' : ''}"
        >
          <ProjectTile name={project.name} id={project.id} class="size-4 rounded text-[10px]" />
          <span class="flex-1 truncate">{project.name}</span>
          {#if waiting > 0}<span class="text-xs text-[var(--brand-text-tertiary)]">{waiting}</span>{/if}
        </a>
      </li>
    {:else}
      <li class="px-2 py-1 text-xs text-[var(--brand-text-tertiary)]">No projects yet</li>
    {/each}
  </ul>
</details>
