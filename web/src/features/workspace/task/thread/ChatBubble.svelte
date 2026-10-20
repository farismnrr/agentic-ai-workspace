<script lang="ts">
  import LogoMark from '../../../../shared/ui/LogoMark.svelte';
  import DocText from '../../components/DocText.svelte';

  interface Props {
    who: 'you' | 'ai';
    text: string;
    time: string;
    /** The teammate speaking, when it is not the lead. */
    agent?: string | undefined;
  }

  const { who, text, time, agent }: Props = $props();
</script>

{#if who === 'you'}
  <div class="flex flex-col items-end gap-1">
    <div class="max-w-[85%] rounded-2xl bg-base-300/60 px-4 py-2"><DocText class="text-[15px] leading-6" {text} /></div>
    <span class="text-xs text-[var(--brand-text-tertiary)]">{time}</span>
  </div>
{:else}
  <div class="flex gap-3">
    <LogoMark class="mt-0.5 size-6 shrink-0" />
    <div class="min-w-0">
      <p class="text-xs text-[var(--brand-text-secondary)]">Masih Awam{agent === undefined ? '' : ` · ${agent}`} · {time}</p>
      <DocText class="mt-0.5 text-[15px] leading-6" {text} />
    </div>
  </div>
{/if}
