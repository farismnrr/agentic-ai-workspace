<script lang="ts">
  import { contextCopy } from '../content/tasks-copy';
  import { contextPercent, contextTone, formatTokens } from '../logic/context-window';

  interface Props {
    used: number;
    window: number;
    /** Show at every width; the header places a second copy on its own row for phones. */
    always?: boolean;
  }

  const { used, window, always = false }: Props = $props();
  const percent = $derived(contextPercent(used, window));
</script>

{#if window > 0}
  <div class="tooltip tooltip-bottom items-center gap-2 text-xs text-[var(--brand-text-secondary)] {always ? 'flex' : 'hidden sm:flex'}" data-tip={contextCopy.tip}>
    <progress class="progress {contextTone(percent)} w-16" value={percent} max="100" aria-label={contextCopy.label}></progress>
    <span class="tabular-nums">{formatTokens(used)} / {formatTokens(window)} · {percent}%</span>
  </div>
{/if}
