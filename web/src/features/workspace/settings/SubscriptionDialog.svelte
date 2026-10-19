<script lang="ts">
  import { programs, subscriptionsCopy } from '../content/subscriptions';
  import {
    describeSignInError, loadSubscription, startSignIn, submitSignInCode, type Program, type SignInPrompt,
  } from '../logic/subscriptions-api';

  interface Props {
    program: Program;
    open: boolean;
    onclose: () => void;
    ondone: () => void;
  }

  const { program, open, onclose, ondone }: Props = $props();
  const copy = $derived(programs[program]);
  let dialog = $state<HTMLDialogElement>();
  let prompt: SignInPrompt | undefined = $state();
  let code = $state('');
  let error = $state('');
  let busy = $state(false);

  async function begin(): Promise<void> {
    error = '';
    busy = true;
    try {
      prompt = await startSignIn(program);
    } catch (failure) {
      error = describeSignInError(failure, subscriptionsCopy.errors);
    } finally {
      busy = false;
    }
  }

  async function poll(): Promise<void> {
    try {
      const state = await loadSubscription(program);
      if (state.connected) ondone();
      else if (state.prompt === undefined) prompt = undefined;
    } catch {
      // The next tick tries again.
    }
  }

  async function finish(event: SubmitEvent): Promise<void> {
    event.preventDefault();
    busy = true;
    error = '';
    try {
      await submitSignInCode(program, code);
      ondone();
    } catch (failure) {
      error = describeSignInError(failure, subscriptionsCopy.errors);
    } finally {
      busy = false;
    }
  }

  $effect(() => {
    if (open && dialog?.open !== true) {
      dialog?.showModal();
      void begin();
    }
    if (!open && dialog?.open === true) dialog.close();
    if (!open) {
      prompt = undefined;
      code = '';
      error = '';
    }
  });

  $effect(() => {
    if (!open || prompt === undefined || program !== 'chatgpt') return undefined;
    const timer = setInterval(() => {
      void poll();
    }, 3000);
    return () => {
      clearInterval(timer);
    };
  });
</script>

<dialog bind:this={dialog} class="modal" onclose={onclose} aria-labelledby="signin-title">
  <form class="modal-box max-w-md" onsubmit={finish}>
    <h2 id="signin-title" class="text-base font-semibold">Sign in to {copy.name}</h2>
    {#if prompt === undefined}
      <p class="mt-3 flex items-center gap-2 text-[var(--brand-text-secondary)]">
        {#if busy}<span class="loading loading-spinner loading-sm" aria-hidden="true"></span>Getting your sign-in page ready…{/if}
      </p>
    {:else}
      <ol class="mt-4 space-y-4">
        <li>
          <p>{copy.steps.open}</p>
          <a class="btn btn-primary btn-sm mt-2" href={prompt.url} target="_blank" rel="noreferrer noopener">Open {copy.name}</a>
        </li>
        <li>
          <p>{copy.steps.finish}</p>
          {#if prompt.code !== undefined}
            <p class="mt-2 select-all rounded-box bg-base-200 px-4 py-3 text-center font-mono text-xl tracking-widest">{prompt.code}</p>
          {:else}
            <label class="sr-only" for="signin-code">Code from {copy.name}</label>
            <input id="signin-code" class="input input-bordered mt-2 w-full" bind:value={code} autocomplete="off" spellcheck="false" required />
          {/if}
        </li>
      </ol>
    {/if}
    {#if error !== ''}<p class="mt-3 text-sm text-error" role="alert">{error}</p>{/if}
    <div class="modal-action">
      <button class="btn btn-sm" type="button" onclick={onclose}>Cancel</button>
      {#if prompt !== undefined && prompt.code === undefined}
        <button class="btn btn-primary btn-sm" type="submit" disabled={busy || code.trim() === ''}>Finish</button>
      {:else if error !== '' && prompt === undefined}
        <button class="btn btn-primary btn-sm" type="button" disabled={busy} onclick={begin}>Try again</button>
      {/if}
    </div>
  </form>
  <form method="dialog" class="modal-backdrop"><button aria-label="Close">close</button></form>
</dialog>
