<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';
  import { proofCopy } from '../../content/tasks-copy';

  interface Props {
    kind: 'receipt' | 'error';
    text: string;
    hint?: string | undefined;
    proven?: boolean | undefined;
    onretry?: (() => void) | undefined;
  }

  const { kind, text, hint, proven = false, onretry }: Props = $props();
  const good = $derived(kind === 'receipt');
</script>

<section class="card border bg-base-100 p-3 {good ? 'border-success/50' : 'border-error/50'}" aria-label={good ? 'Result' : 'Problem'}>
  <p class="flex items-start gap-2 font-medium">
    <Icon name={good ? 'check-circle' : 'alert-triangle'} class="mt-0.5 size-5 shrink-0 {good ? 'text-success' : 'text-error'}" />{text}
  </p>
  {#if good && proven}
    <p class="mt-2 inline-flex items-center gap-1.5 text-sm font-medium text-success"><Icon name="shield" class="size-4" />{proofCopy.proven} · see the Proof tab</p>
  {:else if good}
    <p class="mt-2 inline-flex items-center gap-1.5 text-sm text-[var(--brand-text-secondary)]"><Icon name="info" class="size-4" />{proofCopy.unproven} · see the Proof tab</p>
  {/if}
  {#if hint !== undefined}<p class="mt-2 text-sm text-[var(--brand-text-secondary)]">{hint}</p>{/if}
  {#if onretry !== undefined}<button class="btn btn-outline btn-sm mt-3 rounded-lg" type="button" onclick={onretry}>Continue</button>{/if}
</section>
