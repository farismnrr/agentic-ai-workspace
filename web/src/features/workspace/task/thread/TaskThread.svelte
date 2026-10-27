<script lang="ts">
  import { tick } from 'svelte';
  import type { Task, TaskEvent } from '../../logic/task-types';
  import type { ThreadActions } from '../../logic/task-actions';
  import ActionGroup from './ActionGroup.svelte';
  import ChatBubble from './ChatBubble.svelte';
  import { toBlocks } from './blocks';
  import EventItem from './EventItem.svelte';
  import { shouldShowWorkingIndicator } from '../../logic/working-indicator';
  import { journeySteps } from '../../logic/provenance';

  interface Props {
    task: Task;
    events: TaskEvent[];
    /** Words the AI is writing right now. */
    draft: string;
    actions: ThreadActions;
    onretry?: (() => void) | undefined;
    readOnly?: boolean;
  }

  const { task, events, draft, actions, onretry, readOnly = false }: Props = $props();
  const allBlocks = $derived(toBlocks(events));
  const working = $derived(shouldShowWorkingIndicator(task.status, task.work));
  const journey = $derived(journeySteps(events, task.work));

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

<div bind:this={scroller} {onscroll} class="min-h-0 flex-1 overflow-x-hidden overflow-y-auto px-4 py-6 sm:px-6" role="log" aria-label="Conversation">
  <ol bind:this={content} class="mx-auto flex max-w-3xl flex-col gap-5 wrap-anywhere">
    {#if journey.length > 0}
      <li class="flex flex-wrap items-center gap-1.5 text-xs" aria-label="Work journey">
        <span class="mr-1 text-[var(--brand-text-secondary)]">Journey</span>
        {#each journey as step, index (index)}
          <span class="badge badge-outline badge-sm">{step}</span>
        {/each}
      </li>
    {/if}
    {#if hidden > 0}
      <li class="text-center text-sm text-[var(--brand-text-secondary)]"><button class="link link-hover" type="button" onclick={() => void showEarlier()}>Show {Math.min(hidden, pageSize)} earlier messages</button></li>
    {/if}
    {#each blocks as block (block.kind === 'actions' ? block.id : block.event.id)}
      <li>
        {#if block.kind === 'actions'}
          <div class="ml-9"><ActionGroup items={block.items} onundo={actions.undo} {readOnly} /></div>
        {:else}
          <EventItem event={block.event} taskId={task.id} projectId={task.projectId} provider={task.source} {actions} proof={task.proof} {onretry} {readOnly} />
        {/if}
      </li>
    {/each}
    {#if working && draft !== ''}
      <li><ChatBubble who="ai" text={draft} time="writing…" /></li>
    {:else if working}
      <li class="ml-9 flex items-center gap-2 text-[var(--brand-text-secondary)]" role="status">
        <span class="loading loading-sm motion-safe:loading-dots"></span>{task.status === 'queued' ? 'Waiting to start' : 'Working on it'}
      </li>
    {/if}
  </ol>
</div>
