<script lang="ts">
  import type { EnvVariable } from './env-api';

  interface Props {
    target: EnvVariable;
    password: string;
    message: string;
    revealing: boolean;
    oncancel: () => void;
    onsubmit: () => void;
  }

  let { target, password = $bindable(''), message, revealing, oncancel, onsubmit }: Props = $props();
  let input = $state<HTMLInputElement>();

  $effect(() => {
    input?.focus();
  });
</script>

<dialog class="modal modal-open" aria-labelledby="reveal-title" aria-modal="true" open>
  <div class="modal-box max-w-md">
    <h2 id="reveal-title" class="text-lg font-semibold">Reveal {target.name}</h2>
    <p class="mt-2 text-sm text-[var(--brand-text-secondary)]">Enter your account password to view this saved secret.</p>
    <form class="mt-4 grid gap-4" onsubmit={(event) => { event.preventDefault(); onsubmit(); }}>
      <label class="form-control gap-1" for="reveal-password">
        <span class="label-text">Current account password</span>
        <input bind:this={input} id="reveal-password" class="input input-bordered w-full" type="password"
          bind:value={password} autocomplete="current-password" required />
      </label>
      {#if message}<p class="text-sm text-error" role="alert">{message}</p>{/if}
      <div class="modal-action">
        <button class="btn btn-ghost" type="button" onclick={oncancel}>Cancel</button>
        <button class="btn btn-primary" type="submit" disabled={revealing || password === ''}>
          {revealing ? 'Checking…' : 'Reveal'}
        </button>
      </div>
    </form>
  </div>
  <button class="modal-backdrop" type="button" aria-label="Close reveal dialog" onclick={oncancel}></button>
</dialog>
