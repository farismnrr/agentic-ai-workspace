<script lang="ts">
  import { onMount } from 'svelte';
  import { taskHref } from '../../logic/task-links';
  import {
    approvePlan,
    loadPlanRevision,
    requestPlanChanges,
    type PlanDecisionResponse,
    type PlanRevisionSnapshot,
  } from '../../logic/plan-review-api';
  import { planReviewSnapshotSections } from '../../logic/plan-review-snapshot';
  import { planHandoffPresentation } from '../../logic/work-status';
  import {
    canDecidePlan,
    newPlanDecisionKey,
    planReviewDisplay,
    planReviewLabel,
  } from '../../logic/plan-review-state';
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
  let snapshot = $state<PlanRevisionSnapshot>();
  let inFlight = $state(false);
  let result = $state<PlanDecisionResponse | undefined>();
  let error = $state('');
  let snapshotError = $state('');
  let snapshotLoading = $state(true);
  let confirmingApproval = $state(false);
  let note = $state('');
  const display = $derived(planReviewDisplay(review, result));
  const status = $derived(display.status);
  const label = $derived(planReviewLabel(status));
  const canDecide = $derived(
    result === undefined && snapshot?.revision === review.revision &&
      canDecidePlan(review.status, readOnly, inFlight),
  );
  const linkedTaskId = $derived(display.linkedTaskId);
  const decisionSource = $derived(display.source);
  const decisionTime = $derived(display.decidedAt ?? '');
  const snapshotSections = $derived(snapshot ? planReviewSnapshotSections(snapshot) : []);
  const handoff = $derived(planHandoffPresentation(status, display.handoffState));
  onMount(() => {
    if (review.steps !== undefined) steps = review.steps;
    void loadSnapshot();
  });
  async function loadSnapshot(): Promise<void> {
    snapshotLoading = true;
    snapshotError = '';
    try {
      const loaded = await loadPlanRevision(taskId, review.revision);
      if (loaded.revision !== review.revision) {
        snapshotError = 'The exact plan revision could not be confirmed.';
        return;
      }
      snapshot = loaded;
      if (review.steps === undefined) steps = loaded.steps;
    } catch {
      snapshotError = 'The plan details could not be loaded.';
    } finally {
      snapshotLoading = false;
    }
  }
  async function decide(action: 'approve' | 'changes'): Promise<void> {
    if (action === 'changes') confirmingApproval = false;
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
  function requestApprovalConfirmation(): void {
    if (canDecide) confirmingApproval = true;
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
  <details class="collapse collapse-arrow mt-3 rounded-lg border border-[var(--brand-border)] bg-base-100">
    <summary class="collapse-title min-h-0 py-3 text-sm font-medium">
      Show what the Masih Awam agent team will use
    </summary>
    <div class="collapse-content space-y-4 text-sm">
      {#if snapshot?.revision === review.revision}
        {#each snapshotSections as section (section.label)}
          <section>
            <h3 class="font-semibold">{section.label}</h3>
            <pre class="mt-1 whitespace-pre-wrap break-words font-sans">{section.content}</pre>
          </section>
        {/each}
      {:else if snapshotLoading}
        <p role="status">Loading the plan details…</p>
      {/if}
    </div>
  </details>
  {#if status === 'submitted' && !readOnly}
    <label class="form-control mt-3">
      <span class="label-text text-sm">What should change? <span class="text-base-content/60">(optional)</span></span>
      <textarea class="textarea textarea-bordered mt-1" maxlength="500" bind:value={note}
        disabled={inFlight} placeholder="Add a note about what should change"></textarea>
    </label>
    <p class="mt-3 text-sm">Approving starts the work with the Masih Awam agent team.</p>
    <div class="mt-3 flex flex-wrap gap-2">
      <button class="btn btn-primary btn-sm rounded-lg" type="button" disabled={!canDecide}
        onclick={requestApprovalConfirmation}>
        {inFlight ? 'Saving…' : 'Approve'}
      </button>
      <button class="btn btn-outline btn-sm rounded-lg" type="button" disabled={!canDecide}
        onclick={() => { void decide('changes'); }}>
        Request changes
      </button>
    </div>
    {#if snapshot?.revision !== review.revision}
      <p class="mt-2 text-sm">Approve is disabled until the exact plan details load.</p>
      {#if snapshotError}
        <p class="mt-1 text-sm text-error" role="alert">{snapshotError}</p>
        <button class="btn btn-outline btn-sm mt-2 rounded-lg" type="button"
          disabled={inFlight} onclick={() => { void loadSnapshot(); }}>
          Retry
        </button>
      {/if}
    {/if}
    {#if confirmingApproval}
      <div class="mt-3 rounded-lg border border-warning/40 bg-base-200 p-3" role="group"
        aria-label="Confirm plan approval">
        <p class="text-sm">This will start the approved work. Do you want to continue?</p>
        <div class="mt-3 flex flex-wrap gap-2">
          <button class="btn btn-primary btn-sm rounded-lg" type="button" disabled={!canDecide}
            onclick={() => { void decide('approve'); }}>
            {inFlight ? 'Saving…' : 'Confirm approval'}
          </button>
          <button class="btn btn-ghost btn-sm rounded-lg" type="button" disabled={inFlight}
            onclick={() => { confirmingApproval = false; }}>
            Cancel
          </button>
        </div>
      </div>
    {/if}
  {:else if readOnly && status === 'submitted'}
    <p class="mt-3 text-sm">This task is archived, so the plan can only be viewed.</p>
  {/if}
  {#if status === 'approved'}
    <p class="mt-3 text-sm" role="status">
      {handoff.label}{#if decisionSource} Decision from {decisionSource === 'mcp' ? 'a connected app' : 'the app'}{/if}{#if decisionTime}
        on {new Date(decisionTime).toLocaleString()}
      {/if}.
    </p>
    {#if handoff.state === 'complete' && linkedTaskId}
      <p class="mt-2 text-sm" role="status">
        Build task: <a class="link link-hover font-medium" href={taskHref(projectId, linkedTaskId)}>
          open your build task
        </a>
      </p>
    {/if}
  {:else if status === 'changes_requested'}
    <p class="mt-3 text-sm" role="status">Changes requested. The plan is back with the Masih Awam agent team.</p>
  {/if}
  {#if error}<p class="mt-3 text-sm text-error" role="alert">{error}</p>{/if}
</section>
