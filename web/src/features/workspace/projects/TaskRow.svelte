<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import StatusBadge from '../components/StatusBadge.svelte';
  import StatusDot from '../components/StatusDot.svelte';
  import type { Task } from '../content/task-types';

  interface Props {
    task: Task;
  }

  const { task }: Props = $props();
</script>

<a
  href="/app/projects/{task.projectId}/tasks/{task.id}"
  class="flex min-h-14 items-center gap-3 border-b border-[var(--brand-border)] px-4 py-2 last:border-b-0 hover:bg-base-200/60"
>
  <StatusDot status={task.status} />
  <span class="flex min-w-0 flex-1 flex-col">
    <span class="truncate font-medium">{task.title}</span>
    <span class="truncate text-sm text-[var(--brand-text-secondary)]">{task.summary}</span>
  </span>
  <span class="hidden md:block"><StatusBadge status={task.status} /></span>
  <span class="hidden w-24 text-right text-sm text-[var(--brand-text-tertiary)] sm:block">{task.updated}</span>
  <Icon name="chevron-right" class="size-4 shrink-0 text-[var(--brand-text-tertiary)]" />
</a>
