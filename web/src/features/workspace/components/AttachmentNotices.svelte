<script lang="ts">
  interface Props {
    warning: boolean;
    status: string;
    errors: string[];
  }

  const { warning, status, errors }: Props = $props();
</script>

{#if warning}
  <p class="alert alert-warning alert-soft mt-2 text-sm">
    These files total more than 100 MB and may take a while to upload.
  </p>
{/if}
<p class="mt-2 text-sm text-base-content/70" role="status" aria-atomic="true">{status}</p>
{#if errors.length > 0}
  <ul class="mt-2 list-inside list-disc space-y-1 text-sm text-error" role="alert">
    {#each errors as message, index (index)}<li>{message}</li>{/each}
  </ul>
{/if}
