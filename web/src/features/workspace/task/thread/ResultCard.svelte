<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';
  import DocText from '../../components/DocText.svelte';
  import { proofCopy } from '../../content/tasks-copy';

  interface Props {
    kind: 'receipt' | 'error';
    text: string;
    hint?: string | undefined;
    proven?: boolean | undefined;
    onretry?: (() => void) | undefined;
    /** Opens the Proof tab; without it the proof line is just text. */
    onproof?: (() => void) | undefined;
  }

  const { kind, text, hint, proven = false, onretry, onproof }: Props = $props();
  const good = $derived(kind === 'receipt');
</script>

<section class="card border bg-base-100 p-3 {good ? 'border-success/50' : 'border-error/50'}" aria-label={good ? 'Result' : 'Problem'}>
  <div class="flex items-start gap-2">
    <Icon name={good ? 'check-circle' : 'alert-triangle'} class="mt-0.5 size-5 shrink-0 {good ? 'text-success' : 'text-error'}" />
    <DocText class="flex-1 text-[15px] leading-6" {text} />
  </div>
  {#if good}
    {@const line = `${proven ? proofCopy.proven : proofCopy.unproven} · see the Proof tab`}
    {@const tone = proven ? 'font-medium text-success' : 'text-[var(--brand-text-secondary)]'}
    {#if onproof !== undefined}
      <button type="button" class="mt-2 inline-flex items-center gap-1.5 rounded text-sm underline-offset-2 hover:underline focus-visible:underline {tone}" onclick={onproof}>
        <Icon name={proven ? 'shield' : 'info'} class="size-4" />{line}
      </button>
    {:else}
      <p class="mt-2 inline-flex items-center gap-1.5 text-sm {tone}"><Icon name={proven ? 'shield' : 'info'} class="size-4" />{line}</p>
    {/if}
  {/if}
  {#if hint !== undefined}<p class="mt-2 text-sm text-[var(--brand-text-secondary)]">{hint}</p>{/if}
  {#if onretry !== undefined}<button class="btn btn-outline btn-sm mt-3 rounded-lg" type="button" onclick={onretry}>Continue</button>{/if}
</section>
