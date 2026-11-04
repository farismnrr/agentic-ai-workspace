<script lang="ts">
  import FormField from '../../components/FormField.svelte';
  import { normalizeVariableName } from './env-logic';

  interface Props {
    name: string;
    value: string;
    isSecret: boolean;
    editing: boolean;
    busy: boolean;
    message: string;
    error: boolean;
    onsubmit: () => void;
    oncancel: () => void;
  }

  let {
    name = $bindable(''),
    value = $bindable(''),
    isSecret = $bindable(true),
    editing = $bindable(false),
    busy,
    message,
    error,
    onsubmit,
    oncancel,
  }: Props = $props();

  let valueInput = $state<HTMLInputElement>();

  export function focusValue(): void {
    valueInput?.focus();
  }
</script>

<form class="grid gap-3 py-4" onsubmit={(event) => { event.preventDefault(); onsubmit(); }}>
  <div class="grid gap-3 sm:grid-cols-2">
    <FormField id="variable-name" label="Name" hint="Uppercase letters, numbers and underscores.">
      {#snippet children({ describedby, invalid })}
        <input id="variable-name" class="input input-sm w-full font-mono uppercase" value={name} readonly={editing}
          oninput={(event) => (name = normalizeVariableName(event.currentTarget.value))}
          placeholder="SERVICE_TOKEN" required maxlength="64" pattern="[A-Za-z_][A-Za-z0-9_]*" autocomplete="off" aria-describedby={describedby} aria-invalid={invalid} />
      {/snippet}
    </FormField>
    <FormField id="variable-value" label="Value">
      {#snippet children({ describedby, invalid })}
        <input id="variable-value" bind:this={valueInput} class="input input-sm w-full" type="password" bind:value required
          maxlength="4096" autocomplete="new-password" placeholder="Enter value" aria-describedby={describedby} aria-invalid={invalid} />
      {/snippet}
    </FormField>
  </div>
  <label class="flex w-fit items-center gap-2 text-sm"><input class="checkbox checkbox-sm" type="checkbox" bind:checked={isSecret} />Secret</label>
  <div class="flex flex-wrap items-center gap-3">
    <button class="btn btn-primary btn-sm" type="submit" disabled={busy}>
      {editing ? 'Save replacement' : 'Add variable'}
    </button>
    {#if editing}<button class="btn btn-ghost btn-sm" type="button" onclick={oncancel}>Cancel</button>{/if}
  </div>
  <div class="min-h-6" aria-live="polite">
    {#if message !== ''}<p class="text-xs {error ? 'text-error' : 'text-[var(--brand-text-secondary)]'}" role={error ? 'alert' : 'status'}>{message}</p>{/if}
  </div>
</form>
