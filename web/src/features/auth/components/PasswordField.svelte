<script lang="ts">
  import type { Snippet } from 'svelte';
  import Icon from '../../../shared/ui/Icon.svelte';

  interface Props {
    id: string;
    label: string;
    value?: string;
    autocomplete: 'current-password' | 'new-password';
    placeholder?: string;
    validate: (value: string) => string;
    submitted: boolean;
    labelAside?: Snippet;
    below?: Snippet;
    matched?: boolean;
  }

  let { id, label, value = $bindable(""), autocomplete, placeholder = '', validate, submitted, labelAside, below, matched = false }: Props = $props();

  let touched = $state(false);
  let visible = $state(false);
  const error = $derived(touched || submitted ? validate(value) : '');
</script>

<div class="flex flex-col gap-1.5">
  <div class="flex items-baseline justify-between gap-4">
    <label class="text-sm font-medium" for={id}>{label}</label>
    {@render labelAside?.()}
  </div>
  <div class="input flex h-12 w-full items-center gap-2 pr-2 {error === '' ? '' : 'input-error'}">
    <input
      {id}
      {autocomplete}
      {placeholder}
      type={visible ? 'text' : 'password'}
      bind:value
      class="min-w-0 grow"
      aria-invalid={error === '' ? undefined : 'true'}
      aria-describedby={error === '' ? undefined : `${id}-error`}
      onblur={() => (touched = true)}
    />
    {#if matched}
      <span class="text-success" role="img" aria-label="Matches"><Icon name="check" class="size-5" /></span>
    {/if}
    <button
      type="button"
      class="btn btn-ghost btn-square btn-sm"
      aria-pressed={visible}
      aria-label={visible ? 'Hide password' : 'Show password'}
      onclick={() => (visible = !visible)}
    >
      <Icon name={visible ? 'eye-off' : 'eye'} class="size-5" />
    </button>
  </div>
  {#if error !== ''}
    <p id="{id}-error" class="text-sm text-error">{error}</p>
  {/if}
  {@render below?.()}
</div>
