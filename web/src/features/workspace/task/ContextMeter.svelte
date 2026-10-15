<script lang="ts">
  import { contextCopy } from '../content/tasks-copy';
  import { contextPercent, contextTone, formatTokens } from '../logic/context-window';

  interface Props {
    used: number;
    window: number;
    /** The phone row: no hover on a phone, so the explanation is a visible label instead of a tooltip. */
    phone?: boolean;
  }

  const { used, window, phone = false }: Props = $props();
  const percent = $derived(contextPercent(used, window));
  const size = $derived(`${formatTokens(used)} / ${formatTokens(window)} · ${String(percent)}%`);
</script>

{#if window > 0}
  {#if phone}
    <div class="flex min-w-0 items-center gap-2 text-xs text-[var(--brand-text-secondary)]" title={contextCopy.tip}>
      <progress class="progress {contextTone(percent)} w-16 shrink-0" value={percent} max="100" aria-label={contextCopy.label}></progress>
      <span class="truncate tabular-nums">{contextCopy.label} · {size}</span>
    </div>
  {:else}
    <!-- The tip opens to the left: the meter sits at the right of the header, so it always has room there. -->
    <div class="tooltip tooltip-left hidden items-center gap-2 text-xs text-[var(--brand-text-secondary)] sm:flex" data-tip={contextCopy.hover}>
      <progress class="progress {contextTone(percent)} w-16 shrink-0" value={percent} max="100" aria-label={contextCopy.label}></progress>
      <span class="tabular-nums">{size}</span>
    </div>
  {/if}
{/if}
