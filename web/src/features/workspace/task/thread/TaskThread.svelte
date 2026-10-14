<script lang="ts">
  import type { Task, TaskEvent } from '../../logic/task-types';
  import type { ThreadActions } from '../../logic/task-actions';
  import ActionGroup from './ActionGroup.svelte';
  import { toBlocks } from './blocks';
  import EventItem from './EventItem.svelte';

  interface Props {
    task: Task;
    events: TaskEvent[];
    actions: ThreadActions;
    onretry: () => void;
  }

  const { task, events, actions, onretry }: Props = $props();
  const blocks = $derived(toBlocks(events));
  const working = $derived(['queued', 'planning', 'working'].includes(task.status));
  let scroller = $state<HTMLElement>();

  $effect(() => {
    if (blocks.length > 0 && scroller !== undefined) scroller.scrollTop = scroller.scrollHeight;
  });
</script>

<div bind:this={scroller} class="min-h-0 flex-1 overflow-y-auto px-4 py-6 sm:px-6" role="log" aria-label="Conversation">
  <ol class="mx-auto flex max-w-3xl flex-col gap-5">
    {#each blocks as block (block.kind === 'actions' ? block.id : block.event.id)}
      <li>
        {#if block.kind === 'actions'}
          <div class="ml-9"><ActionGroup items={block.items} onundo={actions.undo} /></div>
        {:else}
          <EventItem event={block.event} {actions} proven={task.proof?.proven ?? false} {onretry} />
        {/if}
      </li>
    {/each}
    {#if working}
      <li class="ml-9 flex items-center gap-2 text-[var(--brand-text-secondary)]" aria-live="polite">
        <span class="loading loading-dots loading-sm"></span>{task.status === 'queued' ? 'Waiting to start' : 'Working on it'}
      </li>
    {/if}
  </ol>
</div>
