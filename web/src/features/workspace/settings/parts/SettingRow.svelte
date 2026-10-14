<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    label: string;
    description?: string | undefined;
    for?: string | undefined;
    children: Snippet;
  }

  const { label, description, for: target, children }: Props = $props();
</script>

<div class="flex flex-wrap items-center justify-between gap-x-6 gap-y-2 py-3">
  <div class="min-w-48 flex-1">
    <label class="font-medium" for={target}>{label}</label>
    {#if description !== undefined}<p class="text-xs text-[var(--brand-text-tertiary)]">{description}</p>{/if}
  </div>
  <div class="flex min-w-0 items-center gap-2">{@render children()}</div>
</div>
