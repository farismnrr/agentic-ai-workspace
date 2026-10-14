<script lang="ts">
  import { route } from '../../../app/router';
  import { dotTone } from '../components/tone';
  import { statusMeta } from '../content/status';
  import { dayGroup, timeAgo, type DayGroup } from '../logic/time';
  import { tasks } from '../logic/workspace-store';

  const activeTask = $derived($route.name === 'task' ? $route.taskId : undefined);
  const recent = $derived($tasks.slice(0, 9));
  const order: DayGroup[] = ['Today', 'Yesterday', 'Earlier'];
  const groups = $derived(
    order.map((label) => ({ label, tasks: recent.filter((task) => dayGroup(task.updatedAt) === label) })).filter((group) => group.tasks.length > 0),
  );
</script>

{#each groups as group (group.label)}
  <p class="px-2 pb-1 pt-3 text-xs text-[var(--brand-text-tertiary)]">{group.label}</p>
  <ul class="flex flex-col">
    {#each group.tasks as task (task.id)}
      {@const meta = statusMeta[task.status]}
      <li>
        <a
          href="/app/projects/{task.projectId}/tasks/{task.id}"
          aria-current={activeTask === task.id ? 'page' : undefined}
          class="group flex items-center gap-2 rounded-lg px-2 py-1.5 hover:bg-base-100/70 {activeTask === task.id ? 'bg-base-300/60 font-medium' : ''}"
        >
          <span class="grid size-2 shrink-0 place-items-center">
            {#if meta.needsYou || task.status === 'working'}<span class="status {dotTone[meta.tone]}"></span>{/if}
          </span>
          <span class="flex-1 truncate">{task.title}</span>
          <span class="shrink-0 text-xs text-[var(--brand-text-tertiary)]">{timeAgo(task.updatedAt).replace(' ago', '')}</span>
        </a>
      </li>
    {/each}
  </ul>
{/each}
