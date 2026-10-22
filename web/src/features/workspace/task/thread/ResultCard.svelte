<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';
  import DocText from '../../components/DocText.svelte';
  import { proofCopy, summarizeChecks } from '../../content/tasks-copy';
  import type { Check } from '../../logic/task-types';

  interface Props {
    kind: 'receipt' | 'error';
    text: string;
    hint?: string | undefined;
    proven?: boolean | undefined;
    checks?: Check[] | undefined;
    onretry?: (() => void) | undefined;
  }

  const { kind, text, hint, proven = false, checks = [], onretry }: Props = $props();
  const good = $derived(kind === 'receipt');
  const checkSummary = $derived(summarizeChecks(checks));
</script>

<section class="card border bg-base-100 p-3 {good ? 'border-success/50' : 'border-error/50'}" aria-label={good ? 'Result' : 'Problem'}>
  <div class="flex items-start gap-2">
    <Icon name={good ? 'check-circle' : 'alert-triangle'} class="mt-0.5 size-5 shrink-0 {good ? 'text-success' : 'text-error'}" />
    <DocText class="flex-1 text-[15px] leading-6" {text} />
  </div>
  {#if good}
    {#if proven}
      <p class="mt-2 inline-flex items-center gap-1.5 text-sm font-medium text-success"><Icon name="shield" class="size-4" />{proofCopy.proven}</p>
    {:else}
      <p class="mt-2 text-sm text-[var(--brand-text-secondary)]">{proofCopy.unproven}</p>
    {/if}
    <p class="mt-1 text-sm text-[var(--brand-text-secondary)]">{checkSummary}</p>
  {/if}
  {#if hint !== undefined}<p class="mt-2 text-sm text-[var(--brand-text-secondary)]">{hint}</p>{/if}
  {#if onretry !== undefined}<button class="btn btn-outline btn-sm mt-3 rounded-lg" type="button" onclick={onretry}>Continue</button>{/if}
</section>
