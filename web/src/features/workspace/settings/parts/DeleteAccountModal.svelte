<script lang="ts">
  import PasswordField from '../../../../shared/forms/PasswordField.svelte';
  import { deleteAccount, describeAccountError } from '../../logic/account-api';

  interface Props {
    open: boolean;
    oncancel: () => void;
    ondeleted: () => void;
  }

  const { open, oncancel, ondeleted }: Props = $props();
  let dialog = $state<HTMLDialogElement>();
  let password = $state('');
  let submitted = $state(false);
  let busy = $state(false);
  let error = $state('');

  const required = (value: string): string => (value === '' ? 'Enter your password to confirm.' : '');

  $effect(() => {
    if (open && dialog?.open !== true) dialog?.showModal();
    if (!open && dialog?.open === true) dialog.close();
  });

  async function onsubmit(event: SubmitEvent): Promise<void> {
    event.preventDefault();
    submitted = true;
    error = '';
    if (required(password) !== '') return;
    busy = true;
    try {
      await deleteAccount(password);
      ondeleted();
    } catch (failure) {
      error = describeAccountError(failure);
    } finally {
      busy = false;
    }
  }
</script>

<dialog bind:this={dialog} class="modal" onclose={oncancel} aria-labelledby="delete-account-title">
  <form class="modal-box flex max-w-md flex-col gap-3" novalidate {onsubmit}>
    <h2 id="delete-account-title" class="text-base font-semibold">Delete your account?</h2>
    <p class="text-[var(--brand-text-secondary)]">
      This removes your account, projects, files, tasks, saved AI keys and connections for good. It cannot be undone. Download your data first if you want a copy.
    </p>
    <PasswordField id="delete-password" label="Your password" autocomplete="current-password" bind:value={password} validate={required} {submitted} />
    {#if error !== ''}<div class="alert alert-error alert-soft text-sm" role="alert">{error}</div>{/if}
    <div class="modal-action">
      <button class="btn btn-sm" type="button" onclick={oncancel}>Keep my account</button>
      <button class="btn btn-error btn-sm" type="submit" disabled={busy}>Delete everything</button>
    </div>
  </form>
  <div class="modal-backdrop"><button type="button" aria-label="Close" onclick={oncancel}>close</button></div>
</dialog>
