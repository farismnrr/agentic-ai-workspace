<script lang="ts">
  import { untrack } from 'svelte';
  import Icon from '../../../../shared/ui/Icon.svelte';
  import { blankDraft, emptyDrafts, isComplete, toAnswers } from '../../logic/question-form';
  import type { Draft } from '../../logic/question-form';
  import type { Question } from '../../logic/task-types';
  import QuestionField from './QuestionField.svelte';

  interface Props {
    questions: Question[];
    answered?: string | undefined;
    /** The teammate asking, when it is not the lead. */
    agent?: string | undefined;
    onanswer: (answers: string[][]) => void;
  }

  const { questions, answered, agent, onanswer }: Props = $props();
  const formId = $props.id();
  // The questions of an event never change, so the drafts start from them once.
  let drafts = $state(untrack(() => emptyDrafts(questions)));

  function submit(event: SubmitEvent): void {
    event.preventDefault();
    if (isComplete(questions, drafts)) onanswer(toAnswers(questions, drafts));
  }
</script>

<section class="card border border-warning/50 bg-base-100 p-3" aria-label="Question for you">
  <p class="flex items-start gap-2 font-medium">
    <Icon name="help-circle" class="mt-0.5 size-5 shrink-0 text-warning" />
    {#if agent !== undefined}<span class="badge badge-outline badge-sm mr-1.5 align-middle">{agent}</span>{/if}
    {questions.length > 1 ? 'A few questions before I start' : (questions[0]?.text ?? '')}
  </p>
  {#if answered !== undefined}
    <p class="mt-3 text-sm font-medium">You answered: {answered}</p>
  {:else}
    <form class="mt-3 flex flex-col gap-4" onsubmit={submit}>
      {#each questions as question, index (index)}
        <QuestionField {question} name={`${formId}-${String(index)}`} single={questions.length === 1} draft={drafts[index] ?? blankDraft} onchange={(next: Draft) => { drafts[index] = next; }} />
      {/each}
      <button class="btn btn-primary btn-sm self-start rounded-lg" type="submit" disabled={!isComplete(questions, drafts)}>Send answers</button>
    </form>
  {/if}
</section>
