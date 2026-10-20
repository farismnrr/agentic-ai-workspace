<script lang="ts">
  import { clock } from '../../logic/time';
  import type { TaskEvent } from '../../logic/task-types';
  import { questionsOf } from '../../logic/question-form';
  import { shotUrl } from '../../logic/tasks-api';
  import type { ThreadActions } from '../../logic/task-actions';
  import ApprovalCard from './ApprovalCard.svelte';
  import ChatBubble from './ChatBubble.svelte';
  import HandoffCard from './HandoffCard.svelte';
  import PlanCard from './PlanCard.svelte';
  import PrdCard from './PrdCard.svelte';
  import QuestionCard from './QuestionCard.svelte';
  import ResultCard from './ResultCard.svelte';
  import ShotCard from './ShotCard.svelte';
  import SiteCard from './SiteCard.svelte';

  interface Props {
    event: TaskEvent;
    taskId: string;
    actions: ThreadActions;
    proven: boolean;
    onretry?: (() => void) | undefined;
    onproof?: (() => void) | undefined;
  }

  const { event, taskId, actions, proven, onretry, onproof }: Props = $props();
</script>

{#if event.kind === 'request'}
  <ChatBubble who="you" text={event.text} time={clock(event.time)} />
{:else if event.kind === 'message'}
  <ChatBubble who="ai" text={event.text} time={clock(event.time)} agent={event.agent} />
{:else if event.kind !== 'action'}
  <div class="ml-9">
    {#if event.kind === 'question'}
      <QuestionCard questions={questionsOf(event)} answered={event.answered} agent={event.agent} onanswer={(answers: string[][]) => { actions.choose(event.id, answers); }} />
    {:else if event.kind === 'approval'}
      <ApprovalCard text={event.text} action={event.action} risk={event.risk} resolved={event.resolved} agent={event.agent} onanswer={(allow: boolean) => { actions.allow(event.id, allow); }} />
    {:else if event.kind === 'error'}
      <ResultCard kind="error" text={event.text} hint={event.hint} {onretry} />
    {:else if event.kind === 'prd' || event.kind === 'research'}
      <PrdCard text={event.text} document={event.technical} agent={event.agent} />
    {:else if event.kind === 'plan'}
      <PlanCard text={event.text} steps={event.technical} />
    {:else if event.kind === 'handoff'}
      <HandoffCard text={event.text} detail={event.technical} state={event.action} agent={event.agent} model={event.model} effort={event.effort} />
    {:else if event.kind === 'shot'}
      <ShotCard text={event.text} src={shotUrl(taskId, event.image)} agent={event.agent} />
    {:else if event.kind === 'site'}
      <SiteCard text={event.text} url={event.url} />
    {:else}
      <ResultCard kind="receipt" text={event.text} {proven} {onproof} />
    {/if}
  </div>
{/if}
