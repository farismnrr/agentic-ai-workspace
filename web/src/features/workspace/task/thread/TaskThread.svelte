<script lang="ts">
  import type { Task } from '../../content/task-types';
  import ActionGroup from './ActionGroup.svelte';
  import { toBlocks } from './blocks';
  import EventItem from './EventItem.svelte';

  interface Props {
    task: Task;
  }

  const { task }: Props = $props();
  const blocks = $derived(toBlocks(task.events));
  let scroller = $state<HTMLElement>();

  $effect(() => {
    if (blocks.length > 0 && scroller !== undefined) scroller.scrollTop = scroller.scrollHeight;
  });
</script>

<div bind:this={scroller} class="min-h-0 flex-1 overflow-y-auto px-4 py-6 sm:px-6" role="log" aria-label="Conversation">
  <ol class="mx-auto flex max-w-3xl flex-col gap-5">
    {#each blocks as block (block.kind === 'actions' ? block.id : block.event.id)}
      <li>
        {#if block.kind === 'actions'}<div class="ml-9"><ActionGroup items={block.items} /></div>{:else}<EventItem event={block.event} />{/if}
      </li>
    {/each}
  </ol>
</div>
