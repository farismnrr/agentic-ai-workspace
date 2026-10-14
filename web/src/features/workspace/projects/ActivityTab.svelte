<script lang="ts">
  import StatusDot from '../components/StatusDot.svelte';
  import { tasksOf } from '../content/tasks';

  interface Props {
    projectId: string;
  }

  const { projectId }: Props = $props();
  const rows = $derived(tasksOf(projectId).flatMap((task) => task.events.filter((event) => event.kind !== 'request' && event.kind !== 'message').map((event) => ({ task, event }))));
</script>

<ul class="card border border-[var(--brand-border)] bg-base-100">
  {#each rows as { task, event } (`${task.id}-${event.id}`)}
    <li class="flex items-center gap-3 border-b border-[var(--brand-border)] px-4 py-3 last:border-b-0">
      <StatusDot status={task.status} />
      <span class="min-w-0 flex-1">
        <span class="block truncate text-sm">{event.text}</span>
        <a class="text-xs text-[var(--brand-text-secondary)] hover:underline" href="/app/projects/{projectId}/tasks/{task.id}">{task.title}</a>
      </span>
      <span class="text-xs text-[var(--brand-text-tertiary)]">{task.started.split(', ')[0]} · {event.time}</span>
    </li>
  {:else}
    <li class="p-5 text-[var(--brand-text-secondary)]">No activity yet.</li>
  {/each}
</ul>
