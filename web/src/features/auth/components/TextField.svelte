<script lang="ts">
  interface Props {
    id: string;
    label: string;
    value?: string;
    type?: 'text' | 'email';
    autocomplete: 'name' | 'email';
    placeholder?: string;
    validate: (value: string) => string;
    submitted: boolean;
  }

  let { id, label, value = $bindable(""), type = 'text', autocomplete, placeholder = '', validate, submitted }: Props = $props();

  let touched = $state(false);
  const error = $derived(touched || submitted ? validate(value) : '');
</script>

<div class="flex flex-col gap-1.5">
  <label class="text-sm font-medium" for={id}>{label}</label>
  <input
    {id}
    {type}
    {autocomplete}
    {placeholder}
    bind:value
    class="input h-12 w-full {error === '' ? '' : 'input-error'}"
    aria-invalid={error === '' ? undefined : 'true'}
    aria-describedby={error === '' ? undefined : `${id}-error`}
    onblur={() => (touched = true)}
  />
  {#if error !== ''}
    <p id="{id}-error" class="text-sm text-error">{error}</p>
  {/if}
</div>
