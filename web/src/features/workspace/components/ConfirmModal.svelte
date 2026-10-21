<script lang="ts">
  interface Props {
    open: boolean;
    title: string;
    body: string;
    warning?: string;
    confirmLabel: string;
    cancelLabel: string;
    oncancel: () => void;
    onconfirm: () => void;
  }

  const { open, title, body, warning = '', confirmLabel, cancelLabel, oncancel, onconfirm }: Props = $props();
  let dialog = $state<HTMLDialogElement>();

  $effect(() => {
    if (open && dialog?.open !== true) dialog?.showModal();
    if (!open && dialog?.open === true) dialog.close();
  });
</script>

<dialog bind:this={dialog} class="modal" onclose={oncancel} aria-labelledby="confirm-title">
  <div class="modal-box max-w-md">
    <h2 id="confirm-title" class="text-base font-semibold">{title}</h2>
    <p class="mt-2 text-[var(--brand-text-secondary)]">{body}</p>
    {#if warning !== ''}<p class="alert alert-warning mt-3 text-sm">{warning}</p>{/if}
    <div class="modal-action">
      <button class="btn btn-sm" type="button" onclick={oncancel}>{cancelLabel}</button>
      <button class="btn btn-error btn-sm" type="button" onclick={onconfirm}>{confirmLabel}</button>
    </div>
  </div>
  <form method="dialog" class="modal-backdrop"><button aria-label="Close">close</button></form>
</dialog>
