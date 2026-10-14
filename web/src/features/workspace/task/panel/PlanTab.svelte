<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';
  import type { PlanState, PlanStep } from '../../logic/task-types';

  interface Props {
    plan: PlanStep[];
  }

  const labels: Record<PlanState, string> = { done: 'Done', active: 'In progress', pending: 'Not started', failed: "Didn't work" };

  const { plan }: Props = $props();
</script>

{#if plan.length === 0}
  <p class="rounded-lg border border-dashed border-[var(--brand-border-strong)] p-4 text-[var(--brand-text-secondary)]">The plan appears here as soon as the AI has made one.</p>
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
        {#if step.detail !== undefined}<p class="mt-1 text-sm text-[var(--brand-text-secondary)]">{step.detail}</p>{/if}
      </div>
    </li>
  {/each}
</ol>
{/if}
