<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';
  import type { ActionEvent } from '../../logic/task-types';
  import { clock } from '../../logic/time';

  interface Props {
    items: ActionEvent[];
    onundo: (eventId: string) => void;
  }

  const { items, onundo }: Props = $props();
  const first = $derived(items[0]?.text ?? '');
  const agent = $derived(items[0]?.agent);
  const summary = $derived(items.length === 1 ? first : `${String(items.length)} steps · ${first}, …`);
</script>

<details class="group">
  <summary class="flex cursor-pointer list-none items-center gap-2 text-[var(--brand-text-secondary)] hover:text-base-content">
    <Icon name="check-circle" class="size-4 shrink-0 text-success" />
    {#if agent !== undefined}<span class="badge badge-outline badge-sm shrink-0">{agent}</span>{/if}
    <span class="truncate">{summary}</span>
    <Icon name="chevron-down" class="size-3.5 shrink-0 transition-transform group-open:rotate-180" />
  </summary>
  <ol class="ml-2 mt-2 flex flex-col gap-2 border-l border-[var(--brand-border)] pl-4">
    {#each items as item (item.id)}
      <li>
        <p>
          {item.text} <span class="text-xs text-[var(--brand-text-tertiary)]">{clock(item.time)}</span>
          {#if item.undone}<span class="badge badge-ghost badge-sm ml-1">Undone</span>
          {:else if item.canUndo}<button class="link link-hover ml-1 text-xs font-medium text-primary" type="button" onclick={() => { onundo(item.id); }}>Undo</button>{/if}
        </p>
        {#if item.technical !== undefined}<code class="mt-1 block wrap-anywhere rounded-md bg-base-200 px-2 py-1 text-xs text-[var(--brand-text-secondary)]">{item.technical}</code>{/if}
      </li>
    {/each}
  </ol>
</details>
