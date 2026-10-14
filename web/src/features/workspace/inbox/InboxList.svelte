<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import StatusDot from '../components/StatusDot.svelte';
  import { findProject } from '../content/projects';
  import type { InboxItem } from '../content/inbox';

  interface Props {
    items: InboxItem[];
    selectedId: string | undefined;
  }

  const { items, selectedId }: Props = $props();
</script>

<ul class="flex flex-col" aria-label="Things waiting for you">
  {#each items as item (item.id)}
    <li class="border-b border-[var(--brand-border)]">
      <a
        href="/app/inbox/{item.id}"
        aria-current={selectedId === item.id ? 'page' : undefined}
        class="flex min-h-16 items-start gap-3 px-4 py-3 hover:bg-base-100/70 {selectedId === item.id ? 'bg-base-100 shadow-[inset_3px_0_0_var(--color-primary)]' : ''}"
      >
        <span class="mt-0.5"><StatusDot status={item.task.status} /></span>
        <span class="flex min-w-0 flex-1 flex-col">
          <span class="truncate text-[15px] font-medium">{item.task.title}</span>
          <span class="truncate text-sm text-[var(--brand-text-secondary)]">{item.headline}</span>
          <span class="mt-0.5 text-xs text-[var(--brand-text-tertiary)]">{findProject(item.task.projectId)?.name} · {item.task.updated}</span>
        </span>
        <Icon name="chevron-right" class="mt-1 size-4 shrink-0 text-[var(--brand-text-tertiary)]" />
      </a>
    </li>
  {/each}
</ul>
