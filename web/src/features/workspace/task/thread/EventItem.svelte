<script lang="ts">
  import { clock } from '../../logic/time';
  import type { TaskEvent } from '../../logic/task-types';
  import type { ThreadActions } from '../../logic/task-actions';
  import ApprovalCard from './ApprovalCard.svelte';
  import ChatBubble from './ChatBubble.svelte';
  import QuestionCard from './QuestionCard.svelte';
  import ResultCard from './ResultCard.svelte';

  interface Props {
    event: TaskEvent;
    actions: ThreadActions;
    proven: boolean;
    onretry?: (() => void) | undefined;
  }

  const { event, actions, proven, onretry }: Props = $props();
</script>

{#if event.kind === 'request'}
  <ChatBubble who="you" text={event.text} time={clock(event.time)} />
{:else if event.kind === 'message'}
  <ChatBubble who="ai" text={event.text} time={clock(event.time)} />
{:else if event.kind !== 'action'}
  <div class="ml-9">
    {#if event.kind === 'question'}
      <QuestionCard text={event.text} options={event.options} answered={event.answered} onanswer={(choice: string) => { actions.choose(event.id, choice); }} />
    {:else if event.kind === 'approval'}
      <ApprovalCard text={event.text} action={event.action} risk={event.risk} resolved={event.resolved} onanswer={(allow: boolean) => { actions.allow(event.id, allow); }} />
    {:else if event.kind === 'error'}
      <ResultCard kind="error" text={event.text} hint={event.hint} {onretry} />
    {:else}
      <ResultCard kind="receipt" text={event.text} {proven} />
    {/if}
  </div>
{/if}
