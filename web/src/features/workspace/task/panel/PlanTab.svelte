<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';
  import EmptyState from '../../../../shared/ui/EmptyState.svelte';
  import DocText from '../../components/DocText.svelte';
  import type { PlanState, PlanStep } from '../../logic/task-types';

  interface Props {
    plan: PlanStep[];
    /** The requirements document the team builds from, in markdown. */
    prd?: string | undefined;
    /** What the team found out on the web before building, in markdown. */
    research?: string | undefined;
  }

  const labels: Record<PlanState, string> = { done: 'Done', active: 'In progress', pending: 'Not started', failed: 'Stuck or failed' };

  const { plan, prd = '', research = '' }: Props = $props();
</script>

<section aria-label="Task plan">
  <h2 class="mb-3 text-sm font-semibold">Plan</h2>
{#if research !== ''}
  <details class="mb-4 rounded-lg border border-[var(--brand-border)] bg-base-100 px-3 py-2">
    <summary class="cursor-pointer font-medium">Research notes</summary>
    <div class="mt-2 max-h-96 overflow-y-auto"><DocText text={research} /></div>
  </details>
{/if}
{#if prd !== ''}
  <details class="mb-4 rounded-lg border border-[var(--brand-border)] bg-base-100 px-3 py-2">
    <summary class="cursor-pointer font-medium">Requirements (PRD)</summary>
    <div class="mt-2 max-h-96 overflow-y-auto"><DocText text={prd} /></div>
  </details>
{/if}
{#if plan.length === 0}
  <EmptyState compact icon="check-square" title="No plan yet" body="The plan appears here as soon as the AI has made one." />
{:else}
<ol class="flex flex-col" aria-label="Plan">
  {#each plan as step, index (step.id)}
    <li class="flex gap-3">
      <div class="flex flex-col items-center">
        {#if step.state === 'done'}<Icon name="check-circle" class="size-6 text-primary" />
        {:else if step.state === 'active'}<Icon name="loader" class="size-6 text-warning motion-safe:animate-spin" />
        {:else if step.state === 'failed'}<Icon name="x-circle" class="size-6 text-error" />
        {:else}<Icon name="clock" class="size-6 text-[var(--brand-text-tertiary)]" />{/if}
        {#if index < plan.length - 1}<span class="my-1 w-0.5 flex-1 {step.state === 'done' ? 'bg-primary/60' : 'bg-base-300'}"></span>{/if}
      </div>
      <div class="pb-5">
        <p class="font-medium leading-6 {step.state === 'pending' ? 'text-[var(--brand-text-secondary)]' : ''}">{step.title}</p>
        <p class="text-xs text-[var(--brand-text-tertiary)]">{labels[step.state]}</p>
        {#if step.detail !== undefined}<DocText class="mt-1 text-sm text-[var(--brand-text-secondary)]" text={step.detail} />{/if}
      </div>
    </li>
  {/each}
</ol>
{/if}
</section>
