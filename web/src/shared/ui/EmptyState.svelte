<script lang="ts">
  import type { Snippet } from 'svelte';
  import Icon from './Icon.svelte';
  import type { IconName } from './icon-names';

  interface Props {
    icon: IconName;
    /** One line that says what is missing; it carries the main information. */
    title: string;
    /** Why it is empty and what to do next. */
    body?: string;
    /** Smaller, for a panel or a list inside a page that has other content. */
    compact?: boolean;
    /** Draws a dashed outline, for an empty area that is not already inside a card. */
    bordered?: boolean;
    /** The one action that resolves the emptiness, when the person can do something about it. */
    children?: Snippet;
    /** Extra classes for the outer box, such as spacing. */
    class?: string;
  }

  const { icon, title, body, compact = false, bordered = false, children, class: className = '' }: Props = $props();
</script>

<div class="{className} flex flex-col items-center text-center {compact ? 'gap-1.5 px-4 py-6' : 'gap-2 px-6 py-14'} {bordered ? 'rounded-xl border border-dashed border-[var(--brand-border-strong)]' : ''}">
  <span class="grid place-items-center rounded-full bg-base-200 text-[var(--brand-text-secondary)] {compact ? 'mb-0.5 size-9' : 'mb-1 size-12'}">
    <Icon name={icon} class={compact ? 'size-[18px]' : 'size-6'} />
  </span>
  <p class="font-medium">{title}</p>
  {#if body !== undefined}<p class="max-w-sm text-sm text-[var(--brand-text-secondary)]">{body}</p>{/if}
  {#if children !== undefined}<div class="mt-2 flex flex-wrap justify-center gap-2">{@render children()}</div>{/if}
</div>
