<script lang="ts">
  import FormField from '../components/FormField.svelte';
  import type { Check } from '../logic/checks-api';
  import { validateCheckCommand, validateCheckName, validateCheckSeconds } from '../logic/check-validation';
  import { joinCommand, splitCommand } from '../logic/command';

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
  let touched = $state<Record<string, boolean>>({});
  let submitted = $state(false);

  const nameError = $derived(touched['name'] === true || submitted ? validateCheckName(name) : '');
  const commandError = $derived(touched['command'] === true || submitted ? validateCheckCommand(command) : '');
  const secondsError = $derived(touched['seconds'] === true || submitted ? validateCheckSeconds(seconds) : '');
  function touch(field: string): void {
    touched = { ...touched, [field]: true };
  }

  function fill(source: Check | undefined): void {
    name = source?.name ?? '';
    description = source?.description ?? '';
    command = joinCommand(source?.command ?? []);
    seconds = source !== undefined && source.timeoutSeconds > 0 ? String(source.timeoutSeconds) : '';
    touched = {};
    submitted = false;
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
    submitted = true;
    if (validateCheckName(name) !== '' || validateCheckCommand(command) !== '' || validateCheckSeconds(seconds) !== '') return;
    onsave({ name: name.trim(), description: description.trim(), command: splitCommand(command), timeoutSeconds: Number(seconds) || 0 });
  }
</script>

<dialog bind:this={dialog} class="modal" onclose={() => (show = false)} aria-labelledby="check-title">
  <form class="modal-box flex max-w-md flex-col gap-4" onsubmit={submit} novalidate>
    <div>
      <h2 id="check-title" class="text-base font-semibold">{check === undefined ? 'Add a check' : 'Edit check'}</h2>
      <p class="mt-1 text-[var(--brand-text-secondary)]">A check is a command the AI can run to prove its work.</p>
    </div>
    <FormField id="check-name" label="Name" hint="Short and lowercase, like test or lint." error={nameError}>
      {#snippet children({ describedby, invalid })}
        <input id="check-name" class="input w-full font-mono {invalid ? 'input-error' : ''}" maxlength="40" placeholder="test" disabled={check !== undefined} aria-invalid={invalid} aria-describedby={describedby} bind:value={name} onblur={() => { touch('name'); }} />
      {/snippet}
    </FormField>
    <FormField id="check-command" label="Command" hint="Separate words with spaces; wrap a word that contains spaces in quotes. It runs without internet access." error={commandError}>
      {#snippet children({ describedby, invalid })}
        <input id="check-command" class="input w-full font-mono {invalid ? 'input-error' : ''}" placeholder="go test ./..." aria-invalid={invalid} aria-describedby={describedby} bind:value={command} onblur={() => { touch('command'); }} />
      {/snippet}
    </FormField>
    <FormField id="check-description" label="What it checks" optional>
      <input id="check-description" class="input w-full" maxlength="120" bind:value={description} />
    </FormField>
    <FormField id="check-seconds" label="Time limit" optional hint="In seconds, up to 600. Without it the check stops after 2 minutes." error={secondsError}>
      {#snippet children({ describedby, invalid })}
        <input id="check-seconds" class="input w-32 {invalid ? 'input-error' : ''}" inputmode="numeric" placeholder="120" aria-invalid={invalid} aria-describedby={describedby} bind:value={seconds} onblur={() => { touch('seconds'); }} />
      {/snippet}
    </FormField>
    {#if error !== ''}<p class="text-sm text-error" role="alert">{error}</p>{/if}
    <div class="modal-action mt-0">
      <button class="btn btn-ghost btn-sm" type="button" onclick={() => (show = false)}>Cancel</button>
      <button class="btn btn-primary btn-sm" type="submit" disabled={busy}>{#if busy}<span class="loading loading-spinner loading-xs"></span>{/if}Save check</button>
    </div>
  </form>
  <button class="modal-backdrop" type="button" aria-label="Close" onclick={() => (show = false)}></button>
</dialog>
