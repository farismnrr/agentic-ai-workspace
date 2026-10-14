<script lang="ts">
  import StatusDot from '../components/StatusDot.svelte';
  import { inboxItems } from '../content/inbox';
  import { findProject } from '../content/projects';

  const items = inboxItems();
  const top = items.slice(0, 4);
</script>

<section aria-labelledby="needs-you">
  <div class="mb-1.5 flex items-baseline justify-between">
    <h2 id="needs-you" class="text-xs font-medium text-[var(--brand-text-secondary)]">Needs you · {items.length}</h2>
    <a class="link link-hover text-xs font-medium text-primary" href="/app/inbox">View all</a>
  </div>
  <ul class="divide-y divide-[var(--brand-border)] rounded-xl border border-[var(--brand-border)] bg-base-100">
    {#each top as item (item.id)}
      <li>
        <a class="flex min-h-11 items-center gap-3 px-3 py-2 hover:bg-base-200/60" href="/app/inbox/{item.id}">
          <StatusDot status={item.task.status} />
          <span class="min-w-0 flex-1 truncate"><span class="font-medium">{item.task.title}</span> <span class="text-[var(--brand-text-secondary)]">· {item.headline}</span></span>
          <span class="hidden text-xs text-[var(--brand-text-tertiary)] sm:block">{findProject(item.task.projectId)?.name}</span>
          <span class="text-xs text-[var(--brand-text-tertiary)]">{item.task.updated}</span>
        </a>
      </li>
    {:else}
      <li class="p-3 text-[var(--brand-text-secondary)]">You're all caught up.</li>
    {/each}
  </ul>
</section>
