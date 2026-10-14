<script lang="ts">
  import Notice from '../components/Notice.svelte';

  let acted = $state(false);
</script>

<div class="mx-auto w-full max-w-[51rem] px-4 pb-3 sm:px-6">
  <div class="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-primary/40 bg-base-100 px-4 py-2.5">
    <p class="text-[var(--brand-text-secondary)]">Nothing is final until you accept.</p>
    <div class="flex gap-2">
      <button class="btn btn-outline btn-sm" type="button" onclick={() => (acted = true)}>Ask for changes</button>
      <button class="btn btn-primary btn-sm" type="button" onclick={() => (acted = true)}>Accept</button>
    </div>
  </div>
  {#if acted}<div class="mt-2"><Notice show /></div>{/if}
</div>
