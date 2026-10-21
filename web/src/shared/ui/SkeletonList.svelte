<script lang="ts">
  interface Props {
    rows?: number;
    variant?: 'project' | 'task' | 'sidebar';
  }

  const { rows = 4, variant = 'task' }: Props = $props();
  const compact = $derived(variant === 'sidebar');
  const withTile = $derived(variant === 'project');
</script>

<ul class="flex flex-col" aria-hidden="true">
  {#each Array.from({ length: rows }).keys() as index (index)}
    <li
      class="flex min-h-14 items-center gap-3 border-b border-[var(--brand-border)] px-4 py-2 last:border-b-0
        {compact ? 'min-h-8 gap-2 px-2 py-1.5' : ''}"
    >
      {#if withTile}
        <span class="skeleton size-9 shrink-0 rounded-lg"></span>
      {:else if !compact}
        <span class="skeleton size-2 shrink-0 rounded-full"></span>
      {/if}
      <span class="flex min-w-0 flex-1 flex-col gap-2">
        {#if index % 2 === 0}
          <span class="skeleton h-3 w-2/3"></span>
        {:else}
          <span class="skeleton h-3 w-1/2"></span>
        {/if}
        {#if !compact}<span class="skeleton h-3 w-1/3"></span>{/if}
      </span>
      {#if compact}
        <span class="skeleton h-3 w-8 shrink-0"></span>
      {:else}
        <span class="skeleton hidden h-3 w-16 shrink-0 sm:block"></span>
      {/if}
    </li>
  {/each}
</ul>
