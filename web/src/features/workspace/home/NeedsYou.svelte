<script lang="ts">
  import StatusDot from '../components/StatusDot.svelte';
  import { homeCopy } from '../content/tasks-copy';
  import { findProject, inboxItems } from '../logic/derive';
  import { timeAgo } from '../logic/time';
  import { projects, tasks } from '../logic/workspace-store';

  const items = $derived(inboxItems($tasks));
</script>

<section aria-labelledby="needs-you">
  <div class="mb-1.5 flex items-baseline justify-between">
    <h2 id="needs-you" class="text-xs font-medium text-[var(--brand-text-secondary)]">Needs you · {items.length}</h2>
    <a class="link link-hover text-xs font-medium text-primary" href="/app/inbox">View all</a>
  </div>
  <ul class="divide-y divide-[var(--brand-border)] rounded-xl border border-[var(--brand-border)] bg-base-100">
    {#each items.slice(0, 4) as item (item.id)}
      <li>
        <a class="flex min-h-11 items-center gap-3 px-3 py-2 hover:bg-base-200/60" href="/app/inbox/{item.id}">
          <StatusDot status={item.task.status} />
          <span class="min-w-0 flex-1 truncate"><span class="font-medium">{item.task.title}</span> <span class="text-[var(--brand-text-secondary)]">· {item.headline}</span></span>
          <span class="hidden text-xs text-[var(--brand-text-tertiary)] sm:block">{findProject($projects, item.task.projectId)?.name}</span>
          <span class="text-xs text-[var(--brand-text-tertiary)]">{timeAgo(item.task.updatedAt)}</span>
        </a>
      </li>
    {:else}
      <li class="p-3 text-[var(--brand-text-secondary)]">{homeCopy.caughtUp}</li>
    {/each}
  </ul>
</section>
