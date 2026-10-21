<script lang="ts">
  import LogoMark from './LogoMark.svelte';
</script>

<div class="grid min-h-dvh place-items-center bg-base-200" role="status" aria-atomic="true">
  <div class="flex flex-col items-center gap-3 text-base-content">
    <LogoMark class="size-12 loading-mark" />
    <span class="text-sm text-[var(--brand-text-secondary)]">Memuat...</span>
  </div>
</div>

<style>
  @media (prefers-reduced-motion: no-preference) {
    :global(.loading-mark) {
      animation: loading-pulse 1.8s ease-in-out infinite;
    }
  }

  @keyframes loading-pulse {
    50% { opacity: 0.68; transform: scale(0.96); }
  }
</style>
