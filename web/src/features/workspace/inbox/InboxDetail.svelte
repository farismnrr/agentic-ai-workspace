<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import Notice from '../components/Notice.svelte';
  import StatusBadge from '../components/StatusBadge.svelte';
  import type { InboxItem } from '../content/inbox';
  import { findProject } from '../content/projects';
  import ApprovalCard from '../task/thread/ApprovalCard.svelte';
  import QuestionCard from '../task/thread/QuestionCard.svelte';
  import ResultCard from '../task/thread/ResultCard.svelte';

  interface Props {
    item: InboxItem;
  }

  const { item }: Props = $props();
  const task = $derived(item.task);
  const href = $derived(`/app/projects/${task.projectId}/tasks/${task.id}`);
  const approval = $derived(task.events.find((event) => event.kind === 'approval'));
  const question = $derived(task.events.find((event) => event.kind === 'question'));
  const problem = $derived(task.events.find((event) => event.kind === 'error'));
  let accepted = $state(false);
</script>

<article class="flex max-w-3xl flex-col gap-5 px-6 py-6">
  <a class="inline-flex items-center gap-2 text-sm text-[var(--brand-text-secondary)] lg:hidden" href="/app/inbox"><Icon name="arrow-left" class="size-4" /> Inbox</a>
  <header>
    <p class="text-sm text-[var(--brand-text-secondary)]">{findProject(task.projectId)?.name} · {item.headline}</p>
    <h2 class="mt-1 text-xl font-semibold">{task.title}</h2>
    <div class="mt-2"><StatusBadge status={task.status} /></div>
  </header>
  {#if item.kind === 'approval' && approval?.kind === 'approval'}
    <ApprovalCard text={approval.text} action={approval.action} risk={approval.risk} resolved={approval.resolved} />
  {:else if item.kind === 'question' && question?.kind === 'question'}
    <QuestionCard text={question.text} options={question.options} answered={question.answered} />
  {:else if item.kind === 'failed' && problem?.kind === 'error'}
    <ResultCard kind="error" text={problem.text} hint={problem.hint} />
  {:else}
    <section class="card border border-[var(--brand-border)] bg-base-100 p-4">
      <p class="font-medium">{task.summary}</p>
      <ul class="mt-3 flex flex-col gap-1.5 text-sm">
        {#each task.proof?.rows ?? [] as row (row.label)}
          <li class="flex justify-between gap-4"><span class="text-[var(--brand-text-secondary)]">{row.label}</span><span>{row.value}</span></li>
        {/each}
      </ul>
      <div class="mt-4 flex gap-2">
        <button class="btn btn-primary btn-sm rounded-lg" type="button" onclick={() => (accepted = true)}>Accept</button>
        <a class="btn btn-outline btn-sm rounded-lg" {href}>See what changed</a>
      </div>
      {#if accepted}<div class="mt-3"><Notice show /></div>{/if}
    </section>
  {/if}
  <a class="link link-hover self-start font-medium text-primary" {href}>Open the whole task</a>
</article>
