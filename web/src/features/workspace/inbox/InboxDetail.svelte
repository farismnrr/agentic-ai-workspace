<script lang="ts">
  import EmptyState from '../../../shared/ui/EmptyState.svelte';
  import { onDestroy, untrack } from 'svelte';
  import type { Readable } from 'svelte/store';
  import Icon from '../../../shared/ui/Icon.svelte';
  import StatusBadge from '../components/StatusBadge.svelte';
  import { findProject, lastProblem, pendingApproval, pendingQuestion, type InboxItem } from '../logic/derive';
  import { createController } from '../logic/task-actions';
  import { questionsOf } from '../logic/question-form';
  import type { Task, TaskEvent } from '../logic/task-types';
  import { createThread } from '../logic/thread';
  import { projects } from '../logic/workspace-store';
  import ApprovalCard from '../task/thread/ApprovalCard.svelte';
  import QuestionCard from '../task/thread/QuestionCard.svelte';
  import ResultCard from '../task/thread/ResultCard.svelte';

  interface Props {
    item: InboxItem;
  }

  const { item }: Props = $props();
  // The parent keys this component by task, so the thread is created once per task.
  const taskId = untrack(() => item.task.id);
  const thread = createThread(taskId);
  const controller = createController(taskId, thread);
  const live: Readable<Task | undefined> = thread.task;
  const events: Readable<TaskEvent[]> = thread.events;
  const error: Readable<string> = controller.error;
  onDestroy(thread.stop);

  const task = $derived($live ?? item.task);
  const href = $derived(`/app/projects/${task.projectId}/tasks/${task.id}`);
  const approval = $derived(pendingApproval($events));
  const question = $derived(pendingQuestion($events));
  const problem = $derived(lastProblem($events));
</script>

<article class="flex max-w-3xl flex-col gap-5 px-6 py-6">
  <a class="inline-flex items-center gap-2 text-sm text-[var(--brand-text-secondary)] lg:hidden" href="/app/inbox"><Icon name="arrow-left" class="size-4" /> Inbox</a>
  <header>
    <p class="text-sm text-[var(--brand-text-secondary)]">{findProject($projects, task.projectId)?.name} · {item.headline}</p>
    <h2 class="mt-1 text-xl font-semibold">{task.title}</h2>
    <div class="mt-2"><StatusBadge status={task.status} /></div>
  </header>
  {#if approval !== undefined}
    <ApprovalCard text={approval.text} action={approval.action} risk={approval.risk} onanswer={(allow: boolean) => { controller.actions.allow(approval.id, allow); }} />
  {:else if question !== undefined}
    <QuestionCard questions={questionsOf(question)} onanswer={(answers: string[][]) => { controller.actions.choose(question.id, answers); }} />
  {:else if task.status === 'failed' && problem !== undefined}
    <ResultCard kind="error" text={problem.text} hint={problem.hint} onretry={controller.retry} />
  {:else if task.status === 'review'}
    <section class="card border border-[var(--brand-border)] bg-base-100 p-4">
      <p class="font-medium">{task.summary}</p>
      <ul class="mt-3 flex flex-col gap-1.5 text-sm">
        {#each task.proof?.rows ?? [] as row (row.label)}
          <li class="flex justify-between gap-4"><span class="text-[var(--brand-text-secondary)]">{row.label}</span><span>{row.value}</span></li>
        {/each}
      </ul>
      <div class="mt-4 flex gap-2">
        <button class="btn btn-primary btn-sm rounded-lg" type="button" onclick={controller.accept}>Accept</button>
        <a class="btn btn-outline btn-sm rounded-lg" {href}>See what changed</a>
      </div>
    </section>
  {:else}
    <EmptyState compact bordered icon="check-circle" title="Nothing is waiting on you here" body="This task no longer needs an answer or your OK." />
  {/if}
  {#if $error !== ''}<p class="text-sm text-error" role="alert">{$error}</p>{/if}
  <a class="link link-hover self-start font-medium text-primary" {href}>Open the whole task</a>
</article>
