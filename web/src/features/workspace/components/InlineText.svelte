<script lang="ts">
  import { parseInline } from '../logic/markdown-inline';

  interface Props {
    /** One line with `code`, **bold**, *italic* and links; shown as elements, never as HTML. */
    text: string;
  }

  const { text }: Props = $props();
</script>

{#each parseInline(text) as span, at (at)}
  {#if span.kind === 'bold'}<strong class="font-semibold">{span.text}</strong>
  {:else if span.kind === 'italic'}<em>{span.text}</em>
  {:else if span.kind === 'code'}<code class="rounded bg-base-200 px-1 py-0.5 font-mono text-[0.85em] break-words">{span.text}</code>
  {:else if span.kind === 'link'}<a class="link break-all" href={span.href} target="_blank" rel="noopener noreferrer">{span.text}</a>
  {:else}{span.text}{/if}
{/each}
