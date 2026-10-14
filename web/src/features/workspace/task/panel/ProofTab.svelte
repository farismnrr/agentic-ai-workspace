<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';
  import type { Proof } from '../../content/task-types';
  import BrowserMock from '../BrowserMock.svelte';

  interface Props {
    proof: Proof | undefined;
  }

  const { proof }: Props = $props();
</script>

{#if proof === undefined}
  <p class="rounded-lg border border-dashed border-[var(--brand-border-strong)] p-4 text-[var(--brand-text-secondary)]">
    Proof shows up here once the work has been tested. You'll see before and after, and what was checked.
  </p>
{:else}
  <div class="grid grid-cols-2 gap-3">
    {#each [{ label: 'Before', message: proof.before, ok: false }, { label: 'After', message: proof.after, ok: true }] as shot (shot.label)}
      <figure>
        <figcaption class="mb-1.5 text-sm text-[var(--brand-text-secondary)]">{shot.label}</figcaption>
        <div class="overflow-hidden rounded-lg border border-[var(--brand-border)]"><BrowserMock message={shot.message} ok={shot.ok} /></div>
      </figure>
    {/each}
  </div>
  <dl class="mt-5 flex flex-col gap-2 text-sm">
    {#each proof.rows as row (row.label)}
      <div class="flex justify-between gap-4 border-b border-dashed border-[var(--brand-border)] pb-2 last:border-b-0">
        <dt class="text-[var(--brand-text-secondary)]">{row.label}</dt>
        <dd class="flex items-center gap-1 text-right {row.label === 'Result' ? 'font-medium text-success' : ''}">
          {row.value}{#if row.label === 'Result'}<Icon name="check" class="size-4" />{/if}
        </dd>
      </div>
    {/each}
  </dl>
  <h3 class="mt-5 font-sans text-sm font-medium">What was checked</h3>
  <ul class="mt-2 flex flex-col gap-1.5 text-sm">
    {#each proof.checks as check (check)}<li class="flex items-start gap-2"><Icon name="check" class="mt-0.5 size-4 shrink-0 text-success" />{check}</li>{/each}
  </ul>
{/if}
