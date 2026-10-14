<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';
  import { proofCopy } from '../../content/tasks-copy';
  import type { Proof } from '../../logic/task-types';

  interface Props {
    proof: Proof | undefined;
  }

  const { proof }: Props = $props();
</script>

{#if proof === undefined}
  <p class="rounded-lg border border-dashed border-[var(--brand-border-strong)] p-4 text-[var(--brand-text-secondary)]">{proofCopy.empty}</p>
{:else}
  <p class="flex items-center gap-2 rounded-lg px-3 py-2 font-medium {proof.proven ? 'bg-success/10 text-success' : 'bg-base-200 text-[var(--brand-text-secondary)]'}">
    <Icon name={proof.proven ? 'shield' : 'info'} class="size-5" />{proof.proven ? proofCopy.proven : proofCopy.unproven}
  </p>
  <dl class="mt-5 flex flex-col gap-2 text-sm">
    {#each proof.rows as row (row.label)}
      <div class="flex justify-between gap-4 border-b border-dashed border-[var(--brand-border)] pb-2 last:border-b-0">
        <dt class="text-[var(--brand-text-secondary)]">{row.label}</dt>
        <dd class="text-right {row.label === 'Result' && proof.proven ? 'font-medium text-success' : ''}">{row.value}</dd>
      </div>
    {/each}
  </dl>
  <h3 class="mt-5 font-sans text-sm font-medium">What was checked</h3>
  <ul class="mt-2 flex flex-col gap-2 text-sm">
    {#each proof.checks as check (check.text)}
      <li class="flex items-start gap-2">
        <Icon name={check.verified ? (check.passed ? 'check-circle' : 'x-circle') : 'info'} class="mt-0.5 size-4 shrink-0 {check.verified ? (check.passed ? 'text-success' : 'text-error') : 'text-[var(--brand-text-tertiary)]'}" />
        <span class="flex-1">{check.text}</span>
        <span class="badge badge-sm {check.verified ? 'badge-success badge-soft' : 'badge-ghost'}">{check.verified ? proofCopy.verified : proofCopy.claimed}</span>
      </li>
    {:else}
      <li class="text-[var(--brand-text-secondary)]">Nothing was checked.</li>
    {/each}
  </ul>
{/if}
