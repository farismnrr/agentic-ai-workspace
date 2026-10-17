<script lang="ts">
  import { parseDoc } from '../logic/markdown';
  import InlineText from './InlineText.svelte';

  interface Props {
    /** Markdown such as the PRD or an answer: headings, lists, tables, code and paragraphs. */
    text: string;
    class?: string;
  }

  const { text, class: classes = 'text-sm leading-6' }: Props = $props();
  const blocks = $derived(parseDoc(text));
  const indent = ['', 'ml-4', 'ml-8'];
</script>

<div class="flex min-w-0 flex-col gap-1.5 break-words {classes}">
  {#each blocks as block, index (index)}
    {#if block.kind === 'heading'}
      <h3 class="font-sans {block.level === 1 ? 'text-base' : 'text-sm'} font-semibold {index === 0 ? '' : 'mt-2'}"><InlineText text={block.text} /></h3>
    {:else if block.kind === 'list'}
      {#if block.ordered}
        <ol class="list-decimal pl-5">{#each block.items as item, at (at)}<li class={indent[item.depth]}><InlineText text={item.text} /></li>{/each}</ol>
      {:else}
        <ul class="list-disc pl-5">{#each block.items as item, at (at)}<li class={indent[item.depth]}><InlineText text={item.text} /></li>{/each}</ul>
      {/if}
    {:else if block.kind === 'code'}
      <pre class="overflow-x-auto rounded-lg bg-base-200 p-3 font-mono text-xs leading-5"><code>{block.text}</code></pre>
    {:else if block.kind === 'table'}
      <div class="overflow-x-auto">
        <table class="table table-sm">
          <thead><tr>{#each block.head as cell, at (at)}<th><InlineText text={cell} /></th>{/each}</tr></thead>
          <tbody>{#each block.rows as row, r (r)}<tr>{#each row as cell, at (at)}<td><InlineText text={cell} /></td>{/each}</tr>{/each}</tbody>
        </table>
      </div>
    {:else if block.kind === 'rule'}
      <hr class="my-1 border-base-300" />
    {:else}
      <p><InlineText text={block.text} /></p>
    {/if}
  {/each}
</div>
