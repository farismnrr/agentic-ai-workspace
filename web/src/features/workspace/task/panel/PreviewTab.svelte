<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';
  import type { Task } from '../../content/task-types';

  interface Props {
    task: Task;
    site: string;
  }

  const { task, site }: Props = $props();
  const changed = $derived(task.changes.length > 0);
</script>

<div class="overflow-hidden rounded-xl border border-[var(--brand-border)] bg-base-100">
  <div class="flex items-center gap-2 border-b border-[var(--brand-border)] bg-base-200 px-3 py-2">
    <Icon name="lock" class="size-3.5 text-[var(--brand-text-tertiary)]" />
    <span class="flex-1 truncate rounded-md bg-base-100 px-2 py-0.5 text-xs">{site}</span>
  </div>
  <div class="flex flex-col gap-3 p-4" aria-hidden="true">
    <div class="h-3 w-24 rounded bg-base-300"></div>
    {#if changed}<div class="rounded-md bg-primary/10 px-3 py-2 text-xs text-primary">Changed: {task.title}</div>{/if}
    <div class="h-24 rounded-md bg-base-200"></div>
    <div class="h-2.5 w-full rounded bg-base-300"></div>
    <div class="h-2.5 w-4/5 rounded bg-base-300"></div>
    <div class="grid grid-cols-3 gap-2"><div class="h-14 rounded-md bg-base-200"></div><div class="h-14 rounded-md bg-base-200"></div><div class="h-14 rounded-md bg-base-200"></div></div>
  </div>
</div>
<p class="mt-3 text-xs text-[var(--brand-text-secondary)]">A sample of {site} {changed ? 'with this task’s changes' : 'as it is now'}. The live preview connects once your site is linked.</p>
