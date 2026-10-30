<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    title: string;
    description?: string | undefined;
    action?: Snippet | undefined;
    children: Snippet;
    /** Keep flat when another component already owns the surface. */
    surface?: boolean;
  }

  const { title, description, action, children, surface = true }: Props = $props();
  const id = $derived(`section-${title.toLowerCase().replaceAll(/\W+/g, '-')}`);
</script>

<section class="mb-5 {surface ? 'rounded-xl border border-[var(--brand-border)] bg-base-100' : ''}" aria-labelledby={id}>
  <div class="flex items-start justify-between gap-3 {surface ? 'border-b border-[var(--brand-border)] px-4 py-3' : 'mb-1'}">
    <div>
      <h2 {id} class="text-sm font-semibold">{title}</h2>
      {#if description !== undefined}<p class="mt-1 text-xs text-[var(--brand-text-secondary)]">{description}</p>{/if}
    </div>
    {#if action !== undefined}{@render action()}{/if}
  </div>
  <div class="divide-y divide-[var(--brand-border)] {surface ? 'px-4' : ''}">{@render children()}</div>
</section>
