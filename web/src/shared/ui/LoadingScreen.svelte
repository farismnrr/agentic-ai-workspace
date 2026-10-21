<script lang="ts">
  const loadingLabel = 'Memuat...';
</script>

<div class="grid min-h-dvh place-items-center bg-base-200 text-base-content" role="status" aria-atomic="true">
  <div class="flex flex-col items-center gap-3">
    <span class="loading loading-spinner loading-md" aria-hidden="true"></span>
    <span class="text-sm text-[var(--brand-text-secondary)]">{loadingLabel}</span>
  </div>
</div>
