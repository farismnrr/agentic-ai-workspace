<script lang="ts">
  import { joinCommand, splitCommand } from '../logic/command';
  import type { Check } from '../logic/checks-api';

  interface Props {
    /** The check being edited, or undefined when adding a new one. */
    check: Check | undefined;
    show?: boolean | undefined;
    error?: string | undefined;
    busy?: boolean | undefined;
    onsave: (check: Check) => void;
  }

  let { check, show = $bindable(false), error = '', busy = false, onsave }: Props = $props();
  let dialog = $state<HTMLDialogElement>();
  let name = $state('');
  let description = $state('');
  let command = $state('');
  let seconds = $state('');

  function fill(source: Check | undefined): void {
    name = source?.name ?? '';
    description = source?.description ?? '';
    command = joinCommand(source?.command ?? []);
    seconds = source !== undefined && source.timeoutSeconds > 0 ? String(source.timeoutSeconds) : '';
  }

  $effect(() => {
    if (show && dialog?.open !== true) {
      fill(check);
      dialog?.showModal();
    }
    if (!show && dialog?.open === true) dialog.close();
  });

  function submit(event: SubmitEvent): void {
    event.preventDefault();
    onsave({ name: name.trim(), description: description.trim(), command: splitCommand(command), timeoutSeconds: Number(seconds) || 0 });
  }
</script>

<dialog bind:this={dialog} class="modal" onclose={() => (show = false)} aria-labelledby="check-title">
  <form class="modal-box max-w-md" onsubmit={submit}>
    <h2 id="check-title" class="text-base font-semibold">{check === undefined ? 'Add a check' : 'Edit check'}</h2>
    <label class="mt-4 block font-medium" for="check-name">Name</label>
    <input id="check-name" class="input input-sm mt-1 w-full font-mono" maxlength="40" placeholder="test" required disabled={check !== undefined} bind:value={name} />
    <label class="mt-4 block font-medium" for="check-command">Command</label>
    <input id="check-command" class="input input-sm mt-1 w-full font-mono" placeholder="go test ./..." required bind:value={command} />
    <p class="mt-1 text-xs text-[var(--brand-text-tertiary)]">Words are separated by spaces. Use quotes for a word that contains spaces.</p>
    <label class="mt-4 block font-medium" for="check-description">What it checks <span class="font-normal text-[var(--brand-text-tertiary)]">(optional)</span></label>
    <input id="check-description" class="input input-sm mt-1 w-full" maxlength="120" bind:value={description} />
    <label class="mt-4 block font-medium" for="check-seconds">Time limit in seconds <span class="font-normal text-[var(--brand-text-tertiary)]">(optional, up to 600)</span></label>
    <input id="check-seconds" class="input input-sm mt-1 w-32" type="number" min="1" max="600" bind:value={seconds} />
    {#if error !== ''}<p class="mt-3 text-sm text-error" role="alert">{error}</p>{/if}
    <div class="modal-action">
      <button class="btn btn-ghost btn-sm" type="button" onclick={() => (show = false)}>Cancel</button>
      <button class="btn btn-primary btn-sm" type="submit" disabled={busy || name.trim() === '' || command.trim() === ''}>Save check</button>
    </div>
  </form>
  <button class="modal-backdrop" type="button" aria-label="Close" onclick={() => (show = false)}></button>
</dialog>
