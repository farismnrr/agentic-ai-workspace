<script lang="ts">
  import { onMount } from 'svelte';
  import { taskHref } from '../../logic/task-links';
  import { approvePlan, loadPlanRevision, requestPlanChanges, type PlanDecisionResponse } from '../../logic/plan-review-api';
  import { canDecidePlan, newPlanDecisionKey, planReviewLabel } from '../../logic/plan-review-state';
  import type { PlanStep } from '../../logic/task-types';

  interface Review {
    revision: number;
    summary: string;
    steps?: PlanStep[];
    status: string;
    source?: string;
    decidedAt?: string;
    handoffState?: string;
    linkedTaskId?: string;
    smallTask?: boolean;
  }

  interface Props {
    review: Review;
    taskId: string;
    projectId: string;
    readOnly?: boolean;
  }

  const { review, taskId, projectId, readOnly = false }: Props = $props();
  let steps = $state<PlanStep[]>([]);
  let inFlight = $state(false);
  let result = $state<PlanDecisionResponse | undefined>();
  let error = $state('');
  let note = $state('');
  const status = $derived(result?.decision ?? review.status);
  const label = $derived(planReviewLabel(status));
  const canDecide = $derived(
    result === undefined && canDecidePlan(review.status, readOnly, inFlight),
  );
  const linkedTaskId = $derived(result?.linkedTaskId ?? review.linkedTaskId);
  const handoffState = $derived(result?.handoffState ?? review.handoffState);
  const decisionSource = $derived(result?.source ?? review.source);
  const decisionTime = $derived(result?.decidedAt ?? review.decidedAt ?? '');

  onMount(() => {
    if (review.steps !== undefined) {
      steps = review.steps;
      return;
    }
    if (steps.length > 0) return;
    void loadPlanRevision(taskId, review.revision)
      .then((snapshot) => (steps = snapshot.steps))
      .catch(() => (error = 'The plan details could not be loaded.'));
  });

  async function decide(action: 'approve' | 'changes'): Promise<void> {
    inFlight = true;
    error = '';
    const key = newPlanDecisionKey();
    try {
      if (action === 'approve') {
        result = await approvePlan(taskId, review.revision, key);
      } else {
        result = await requestPlanChanges(taskId, review.revision, key, note);
      }
    } catch {
      error = 'That choice could not be saved. Refresh the task and try again.';
    } finally {
      inFlight = false;
    }
  }
</script>

<section class="card border border-[var(--brand-border)] bg-base-100 p-4" aria-label={label}>
  {#if status === 'submitted'}
    <p class="sr-only" role="status">{label}, revision {review.revision}</p>
  {/if}
  <div class="flex flex-wrap items-center gap-2">
    <span class="badge badge-primary">{label}</span>
    {#if review.smallTask}<span class="badge badge-secondary">Small task</span>{/if}
    <span class="text-sm text-[var(--brand-text-secondary)]">Revision {review.revision}</span>
  </div>
  <p class="mt-3 font-medium">{review.summary}</p>
  {#if steps.length > 0}
    <ol class="mt-2 list-inside list-decimal space-y-1 text-sm">
      {#each steps as step (step.id)}
        <li><span class="font-medium">{step.title}</span>{#if step.detail}<span> — {step.detail}</span>{/if}</li>
      {/each}
    </ol>
  {/if}
  {#if status === 'submitted' && !readOnly}
    <label class="form-control mt-3">
      <span class="label-text text-sm">What should change? <span class="text-base-content/60">(optional)</span></span>
      <textarea class="textarea textarea-bordered mt-1" maxlength="500" bind:value={note}
        disabled={inFlight} placeholder="Add a note for the planner"></textarea>
    </label>
    <div class="mt-3 flex flex-wrap gap-2">
      <button class="btn btn-primary btn-sm rounded-lg" type="button" disabled={!canDecide}
        onclick={() => { void decide('approve'); }}>
        {inFlight ? 'Saving…' : 'Approve'}
      </button>
      <button class="btn btn-outline btn-sm rounded-lg" type="button" disabled={!canDecide}
        onclick={() => { void decide('changes'); }}>
        Request changes
      </button>
    </div>
  {:else if readOnly && status === 'submitted'}
    <p class="mt-3 text-sm">This task is archived, so the plan can only be viewed.</p>
  {/if}
  {#if status === 'approved'}
    <p class="mt-3 text-sm" role="status">
      Approved{#if decisionSource} from {decisionSource === 'mcp' ? 'a connected app' : 'the app'}{/if}{#if decisionTime}
        on {new Date(decisionTime).toLocaleString()}
      {/if}.
    </p>
    {#if linkedTaskId}
      <p class="mt-2 text-sm" role="status">
        Handed off: <a class="link link-hover font-medium" href={taskHref(projectId, linkedTaskId)}>
          open the internal task
        </a>
      </p>
    {:else if handoffState === 'pending' || handoffState === 'created'}
      <p class="mt-2 text-sm" role="status">Handed off. The internal task is being prepared.</p>
    {:else if handoffState === 'failed'}
      <p class="mt-2 text-sm" role="status">Approval saved, but the handoff needs attention.</p>
    {/if}
  {:else if status === 'changes_requested'}
    <p class="mt-3 text-sm" role="status">Changes requested. The plan is back with the planner.</p>
  {/if}
  {#if error}<p class="mt-3 text-sm text-error" role="alert">{error}</p>{/if}
</section>
