<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import { kindChoices } from '../content/models';
  import { describeProviderError } from '../logic/provider-errors';
  import { addProvider, verifyProvider, type ProviderKind } from '../logic/providers-api';

  interface Props {
    show?: boolean | undefined;
    onadded: (verified: boolean) => void;
  }

  let { show = $bindable(false), onadded }: Props = $props();
  let dialog = $state<HTMLDialogElement>();
  let kind: ProviderKind = $state('anthropic');
  let name = $state('Anthropic');
  let baseUrl = $state('');
  let key = $state('');
  let shown = $state(false);
  let busy = $state(false);
  let error = $state('');

  const choice = $derived(kindChoices.find((item) => item.value === kind) ?? kindChoices[0]);

  $effect(() => {
    if (show && dialog?.open !== true) dialog?.showModal();
    if (!show && dialog?.open === true) dialog.close();
  });

  function pick(): void {
    name = choice?.name ?? '';
  }

  async function submit(event: SubmitEvent): Promise<void> {
    event.preventDefault();
    busy = true;
    error = '';
    try {
      const created = await addProvider({ kind, name, baseUrl, apiKey: key });
      let verified = true;
      try {
        await verifyProvider(created.id);
      } catch {
        verified = false;
      }
      key = '';
      show = false;
      onadded(verified);
    } catch (failure) {
      error = describeProviderError(failure);
    } finally {
      busy = false;
    }
  }
</script>

<dialog bind:this={dialog} class="modal" onclose={() => (show = false)} aria-labelledby="add-provider-title">
  <form class="modal-box max-w-md" onsubmit={submit}>
    <h2 id="add-provider-title" class="text-base font-semibold">Add provider</h2>
    <label class="mt-4 block font-medium" for="provider-kind">Provider</label>
    <select id="provider-kind" class="select select-sm mt-1 w-full" bind:value={kind} onchange={pick}>
      {#each kindChoices as option (option.value)}<option value={option.value}>{option.label}</option>{/each}
    </select>
    <label class="mt-4 block font-medium" for="provider-name">Name</label>
    <input id="provider-name" class="input input-sm mt-1 w-full" maxlength="60" bind:value={name} />
    <label class="mt-4 block font-medium" for="provider-url">Address {kind === 'compatible' ? '' : '(optional)'}</label>
    <input id="provider-url" class="input input-sm mt-1 w-full font-mono text-xs" placeholder={choice?.baseUrlHint ?? ''} bind:value={baseUrl} required={kind === 'compatible'} />
    <label class="mt-4 block font-medium" for="api-key">API key</label>
    <div class="relative mt-1">
      <input id="api-key" class="input input-sm w-full pr-10" type={shown ? 'text' : 'password'} autocomplete="off" required bind:value={key} />
      <button class="btn btn-ghost btn-xs btn-square absolute right-1 top-1" type="button" aria-label={shown ? 'Hide key' : 'Show key'} onclick={() => (shown = !shown)}>
        <Icon name={shown ? 'eye-off' : 'eye'} class="size-4" />
      </button>
    </div>
    <p class="mt-3 flex gap-2 text-xs text-[var(--brand-text-secondary)]"><Icon name="lock" class="mt-0.5 size-3.5 shrink-0" /> Your key is checked with the provider and stored encrypted. It is never shown again in full.</p>
    {#if choice?.keyHelp}<a class="link link-hover mt-3 inline-block text-xs font-medium text-primary" href={choice.keyHelp} target="_blank" rel="noreferrer">Where do I get a key?</a>{/if}
    {#if error !== ''}<p class="mt-3 text-sm text-error" role="alert">{error}</p>{/if}
    <div class="modal-action">
      <button class="btn btn-ghost btn-sm" type="button" onclick={() => (show = false)}>Cancel</button>
      <button class="btn btn-primary btn-sm" type="submit" disabled={busy || name.trim() === '' || key.trim() === ''}>Save and verify</button>
    </div>
  </form>
  <button class="modal-backdrop" type="button" aria-label="Close" onclick={() => (show = false)}></button>
</dialog>
