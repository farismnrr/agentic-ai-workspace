<script lang="ts">
  import { dayGroup, type DayGroup } from '../logic/time';
  import { loaded, tasks } from '../logic/workspace-store';
  import SkeletonList from '../../../shared/ui/SkeletonList.svelte';
  import SidebarTaskRow from './SidebarTaskRow.svelte';

  const recent = $derived($tasks.filter((task) => task.pinnedAt === null).slice(0, 9));
  const order: DayGroup[] = ['Today', 'Yesterday', 'Earlier'];
  const groups = $derived(
    order.map((label) => ({ label, tasks: recent.filter((task) => dayGroup(task.updatedAt) === label) })).filter((group) => group.tasks.length > 0),
  );
</script>

{#if !$loaded}
  <p class="px-2 pb-1 pt-3 text-xs text-[var(--brand-text-tertiary)]">Recent</p>
  <SkeletonList variant="sidebar" rows={5} />
{:else}
{#each groups as group (group.label)}
  <p class="px-2 pb-1 pt-3 text-xs text-[var(--brand-text-tertiary)]">{group.label}</p>
  <ul class="flex flex-col">
    {#each group.tasks as task (task.id)}
      <SidebarTaskRow {task} />
    {/each}
  </ul>
{/each}
{/if}
