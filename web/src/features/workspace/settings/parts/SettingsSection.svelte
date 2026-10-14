<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    title: string;
    description?: string | undefined;
    action?: Snippet | undefined;
    children: Snippet;
  }

  const { title, description, action, children }: Props = $props();
  const id = $derived(`section-${title.toLowerCase().replaceAll(/\W+/g, '-')}`);
</script>

<section class="mb-8" aria-labelledby={id}>
  <div class="mb-1 flex items-start justify-between gap-3">
    <div>
      <h2 {id} class="text-sm font-semibold">{title}</h2>
      {#if description !== undefined}<p class="mt-0.5 text-xs text-[var(--brand-text-tertiary)]">{description}</p>{/if}
    </div>
    {#if action !== undefined}{@render action()}{/if}
  </div>
  <div class="divide-y divide-[var(--brand-border)]">{@render children()}</div>
</section>
