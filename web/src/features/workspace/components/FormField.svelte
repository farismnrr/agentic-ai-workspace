<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    id: string;
    label: string;
    optional?: boolean | undefined;
    hint?: string | undefined;
    hintLink?: { href: string; label: string } | undefined;
    /** Shown under the field instead of the hint, once the person has touched it or tried to submit. */
    error?: string | undefined;
    children: Snippet<[{ describedby: string | undefined; invalid: boolean }]>;
  }

  const { id, label, optional = false, hint, hintLink, error = '', children }: Props = $props();
  const invalid = $derived(error !== '');
  const hasHint = $derived(hint !== undefined);
  const describedby = $derived(invalid ? `${id}-error` : !hasHint ? undefined : `${id}-hint`);
</script>

<div class="flex flex-col gap-1">
  <label class="flex items-baseline gap-1 font-medium" for={id}>{label}{#if optional}<span class="font-normal text-[var(--brand-text-secondary)]">(optional)</span>{/if}</label>
  {@render children({ describedby, invalid })}
  {#if invalid}
    <p id="{id}-error" class="text-sm text-error">{error}</p>
  {:else if hasHint}
    <p id="{id}-hint" class="text-xs text-[var(--brand-text-secondary)]">{hint}{#if hintLink !== undefined}&nbsp;<a class="link link-primary" href={hintLink.href}>{hintLink.label}</a>.{/if}</p>
  {/if}
</div>
