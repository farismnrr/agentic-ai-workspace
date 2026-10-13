<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import type { ChecklistItem } from '../content';

  interface Props {
    items: ChecklistItem[];
    title?: string;
    class?: string;
  }

  const { items, title, class: className = '' }: Props = $props();
</script>

<div class="card border border-[var(--brand-border)] bg-base-100 shadow-[var(--brand-shadow)] {className}">
  <div class="card-body gap-3 p-4 text-sm sm:p-5">
    {#if title !== undefined}
      <p class="flex items-start gap-3 text-base font-medium leading-snug">
        <Icon name="chat" class="mt-0.5 size-5 shrink-0" />
        {title}
      </p>
    {/if}
    <ul class="flex flex-col gap-2.5">
      {#each items as item (item.label)}
        <li class="flex items-center gap-3">
          <span
            class="grid size-5 shrink-0 place-items-center rounded-full border {item.status === 'Done'
              ? 'border-primary bg-[var(--brand-marker)] text-primary'
              : 'border-[var(--brand-border-strong)] text-[var(--brand-text-tertiary)]'}"
          >
            <Icon name="check" class="size-3" />
          </span>
          <span>{item.label}: <span class="text-[var(--brand-text-secondary)]">{item.status}</span></span>
        </li>
      {/each}
    </ul>
  </div>
</div>
