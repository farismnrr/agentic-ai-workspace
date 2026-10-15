<script lang="ts">
  import { contextCopy } from '../content/tasks-copy';
  import { contextPercent, contextTone, formatTokens } from '../logic/context-window';

  interface Props {
    used: number;
    window: number;
  }

  const { used, window }: Props = $props();
  const percent = $derived(contextPercent(used, window));
</script>

{#if window > 0}
  <div class="tooltip tooltip-bottom hidden items-center gap-2 text-xs text-[var(--brand-text-secondary)] sm:flex" data-tip={contextCopy.tip}>
    <progress class="progress {contextTone(percent)} w-16" value={percent} max="100" aria-label={contextCopy.label}></progress>
    <span class="tabular-nums">{formatTokens(used)} / {formatTokens(window)} · {percent}%</span>
  </div>
{/if}
