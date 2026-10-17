<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';

  interface Props {
    text: string;
    src: string;
    /** The teammate who looked, when it is not the lead. */
    agent?: string | undefined;
  }

  const { text, src, agent }: Props = $props();
</script>

<figure class="card overflow-hidden border border-base-300 bg-base-100" aria-label="Screenshot of your app">
  <a href={src} target="_blank" rel="noopener noreferrer" class="block bg-base-200">
    <img class="max-h-96 w-full object-cover object-top" {src} alt={text} loading="lazy" />
  </a>
  <figcaption class="flex items-center gap-2 p-3 text-sm">
    <Icon name="globe" class="size-4 shrink-0 text-[var(--brand-text-secondary)]" />
    {#if agent !== undefined}<span class="badge badge-outline badge-sm">{agent}</span>{/if}
    <span class="min-w-0 flex-1">{text}</span>
    <a class="link text-xs" href={src} target="_blank" rel="noopener noreferrer">Open full size</a>
  </figcaption>
</figure>
