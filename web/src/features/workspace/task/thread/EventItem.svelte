<script lang="ts">
  import type { TaskEvent } from '../../content/task-types';
  import ApprovalCard from './ApprovalCard.svelte';
  import ChatBubble from './ChatBubble.svelte';
  import QuestionCard from './QuestionCard.svelte';
  import ResultCard from './ResultCard.svelte';

  interface Props {
    event: TaskEvent;
  }

  const { event }: Props = $props();
</script>

{#if event.kind === 'request'}
  <ChatBubble who="you" text={event.text} time={event.time} />
{:else if event.kind === 'message'}
  <ChatBubble who="ai" text={event.text} time={event.time} />
{:else}
  <div class="ml-9">
    {#if event.kind === 'question'}
      <QuestionCard text={event.text} options={event.options} answered={event.answered} />
    {:else if event.kind === 'approval'}
      <ApprovalCard text={event.text} action={event.action} risk={event.risk} resolved={event.resolved} />
    {:else if event.kind === 'error'}
      <ResultCard kind="error" text={event.text} hint={event.hint} />
    {:else if event.kind === 'receipt'}
      <ResultCard kind="receipt" text={event.text} />
    {/if}
  </div>
{/if}
