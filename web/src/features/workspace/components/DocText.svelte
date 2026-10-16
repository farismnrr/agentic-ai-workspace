<script lang="ts">
  import { parseDoc } from '../logic/prd';

  interface Props {
    /** Markdown with headings, lists and paragraphs, such as the PRD. */
    text: string;
  }

  const { text }: Props = $props();
  const blocks = $derived(parseDoc(text));
</script>

<div class="flex flex-col gap-1.5 text-sm leading-6">
  {#each blocks as block, index (index)}
    {#if block.kind === 'heading'}
      <h3 class="font-sans {block.level === 1 ? 'text-base' : 'text-sm'} font-semibold {index === 0 ? '' : 'mt-2'}">{block.text}</h3>
    {:else if block.kind === 'list'}
      {#if block.ordered}
        <ol class="list-decimal pl-5">{#each block.items as item, at (at)}<li>{item}</li>{/each}</ol>
      {:else}
        <ul class="list-disc pl-5">{#each block.items as item, at (at)}<li>{item}</li>{/each}</ul>
      {/if}
    {:else}
      <p>{block.text}</p>
    {/if}
  {/each}
</div>
