<script lang="ts">
  interface Props {
    saving: boolean;
    message: string;
    failed?: boolean | undefined;
    onsave: () => void;
  }

  const { saving, message, failed = false, onsave }: Props = $props();
</script>

<div class="sticky bottom-0 -mx-1 flex flex-col items-end gap-2 border-t border-[var(--brand-border)] bg-base-200/95 px-1 py-3 backdrop-blur">
  <button class="btn btn-primary btn-sm rounded-lg" type="button" disabled={saving} onclick={onsave}>Save changes</button>
  {#if message !== ''}<p class="text-xs {failed ? 'text-error' : 'text-[var(--brand-text-secondary)]'}" role="status">{message}</p>{/if}
</div>
