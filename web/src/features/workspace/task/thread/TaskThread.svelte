<script lang="ts">
  import { tick } from 'svelte';
  import type { Task, TaskEvent } from '../../logic/task-types';
  import type { ThreadActions } from '../../logic/task-actions';
  import ActionGroup from './ActionGroup.svelte';
  import ChatBubble from './ChatBubble.svelte';
  import { toBlocks } from './blocks';
  import EventItem from './EventItem.svelte';

  interface Props {
    task: Task;
    events: TaskEvent[];
    /** Words the AI is writing right now. */
    draft: string;
    actions: ThreadActions;
    onretry: () => void;
  }

  const { task, events, draft, actions, onretry }: Props = $props();
  const allBlocks = $derived(toBlocks(events));
  const working = $derived(['queued', 'planning', 'working'].includes(task.status));

  // Only the latest messages are drawn; older ones are added as the person scrolls up to them.
  const pageSize = 40;
  const nearBottom = 96;
  const nearTop = 160;
  let shown = $state(pageSize);
  const blocks = $derived(allBlocks.slice(-shown));
  const hidden = $derived(allBlocks.length - blocks.length);

  let scroller = $state<HTMLElement>();
  let content = $state<HTMLElement>();
  let following = true;

  const toBottom = (): void => {
    if (scroller !== undefined) scroller.scrollTop = scroller.scrollHeight;
  };

  async function showEarlier(): Promise<void> {
    const box = scroller;
    if (box === undefined) return;
    const before = box.scrollHeight;
    shown += pageSize;
    await tick();
    // Keep the message the person was reading where it was.
    box.scrollTop += box.scrollHeight - before;
  }

  function onscroll(): void {
    if (scroller === undefined) return;
    following = scroller.scrollHeight - scroller.scrollTop - scroller.clientHeight < nearBottom;
    if (scroller.scrollTop < nearTop && hidden > 0) void showEarlier();
  }

  $effect(() => {
    // Open at the latest message and stay there while the conversation grows, including words
    // still being written and content that settles later (fonts, formatting), until the person
    // scrolls up to read; scrolling back to the end resumes following.
    if (content === undefined) return;
    const watcher = new ResizeObserver(() => {
      if (following) toBottom();
    });
    watcher.observe(content);
    toBottom();
    return () => {
      watcher.disconnect();
    };
  });
</script>

<div bind:this={scroller} {onscroll} class="min-h-0 flex-1 overflow-y-auto px-4 py-6 sm:px-6" role="log" aria-label="Conversation">
  <ol bind:this={content} class="mx-auto flex max-w-3xl flex-col gap-5">
    {#if hidden > 0}
      <li class="text-center text-sm text-[var(--brand-text-secondary)]"><button class="link link-hover" type="button" onclick={() => void showEarlier()}>Show {Math.min(hidden, pageSize)} earlier messages</button></li>
    {/if}
    {#each blocks as block (block.kind === 'actions' ? block.id : block.event.id)}
      <li>
        {#if block.kind === 'actions'}
          <div class="ml-9"><ActionGroup items={block.items} onundo={actions.undo} /></div>
        {:else}
          <EventItem event={block.event} {actions} proven={task.proof?.proven ?? false} {onretry} />
        {/if}
      </li>
    {/each}
    {#if working && draft !== ''}
      <li><ChatBubble who="ai" text={draft} time="writing…" /></li>
    {:else if working}
      <li class="ml-9 flex items-center gap-2 text-[var(--brand-text-secondary)]" aria-live="polite">
        <span class="loading loading-dots loading-sm"></span>{task.status === 'queued' ? 'Waiting to start' : 'Working on it'}
      </li>
    {/if}
  </ol>
</div>
