<script lang="ts">
  import { rawUrl } from '../../logic/files-api';
  import { clock } from '../../logic/time';
  import type { Proof, TaskEvent } from '../../logic/task-types';
  import { splitAttachments } from '../../logic/message-attachments';
  import { questionsOf } from '../../logic/question-form';
  import { shotUrl } from '../../logic/tasks-api';
  import type { ThreadActions } from '../../logic/task-actions';
  import ApprovalCard from './ApprovalCard.svelte';
  import ChatBubble from './ChatBubble.svelte';
  import ExtensionCard from './ExtensionCard.svelte';
  import HandoffCard from './HandoffCard.svelte';
  import PlanCard from './PlanCard.svelte';
  import PrdCard from './PrdCard.svelte';
  import QuestionCard from './QuestionCard.svelte';
  import SentFiles from './SentFiles.svelte';
  import ResultCard from './ResultCard.svelte';
  import ShotCard from './ShotCard.svelte';
  import SiteCard from './SiteCard.svelte';

  interface Props {
    event: TaskEvent;
    taskId: string;
    projectId: string;
    actions: ThreadActions;
    proof?: Proof | undefined;
    onretry?: (() => void) | undefined;
    readOnly?: boolean;
  }

  const { event, taskId, projectId, actions, proof, onretry, readOnly = false }: Props = $props();
</script>

{#if event.kind === 'request'}
  {@const sent = splitAttachments(event.text)}
  {#if sent.files.length > 0}
    <div class="mb-1"><SentFiles {projectId} files={sent.files} /></div>
  {/if}
  {#if sent.text !== '' || sent.files.length === 0}
    <ChatBubble who="you" text={sent.text} time={clock(event.time)} />
  {/if}
  {#if event.documents?.[0] !== undefined}
    <div class="mt-2 flex justify-end"><a class="badge badge-outline link" href={rawUrl(projectId, event.documents[0])}>Download full message</a></div>
  {/if}
{:else if event.kind === 'message'}
  <ChatBubble who="ai" text={event.text} time={clock(event.time)} agent={event.agent} />
{:else if event.kind !== 'action'}
  <div class="ml-9">
    {#if event.kind === 'question'}
      <QuestionCard questions={questionsOf(event)} answered={event.answered} agent={event.agent} {readOnly} onanswer={(answers: string[][]) => { actions.choose(event.id, answers); }} />
    {:else if event.kind === 'approval'}
      <ApprovalCard text={event.text} action={event.action} risk={event.risk} resolved={event.resolved} agent={event.agent} {readOnly} onanswer={(allow: boolean) => { actions.allow(event.id, allow); }} />
    {:else if event.kind === 'error'}
      <ResultCard kind="error" text={event.text} hint={event.hint} onretry={readOnly ? undefined : onretry} />
    {:else if event.kind === 'prd' || event.kind === 'research'}
      <PrdCard text={event.text} document={event.technical} agent={event.agent} />
    {:else if event.kind === 'plan'}
      <PlanCard text={event.text} steps={event.technical} />
    {:else if event.kind === 'handoff'}
      <HandoffCard text={event.text} detail={event.technical} state={event.action} agent={event.agent} model={event.model} effort={event.effort} />
    {:else if event.kind === 'shot'}
      <ShotCard text={event.text} src={shotUrl(taskId, event.image)} agent={event.agent} />
    {:else if event.kind === 'extension'}
      <ExtensionCard text={event.text} url={event.url} />
    {:else if event.kind === 'site'}
      <SiteCard text={event.text} url={event.url} />
    {:else}
      <ResultCard kind="receipt" text={event.text} proven={proof?.proven ?? false} checks={proof?.checks ?? []} />
    {/if}
  </div>
{/if}
