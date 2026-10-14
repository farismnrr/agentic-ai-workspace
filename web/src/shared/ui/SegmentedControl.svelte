<script lang="ts">
  interface Option {
    value: string;
    label: string;
  }

  interface Props {
    name: string;
    legend: string;
    options: Option[];
    value?: string | undefined;
    disabled?: boolean;
    class?: string;
  }

  let { name, legend, options, value = $bindable(''), disabled = false, class: className = '' }: Props = $props();
</script>

<fieldset class="join {className}" {disabled}>
  <legend class="sr-only">{legend}</legend>
  {#each options as option (option.value)}
    <input
      class="btn join-item btn-sm rounded-full border-base-300 bg-base-200 shadow-none checked:border-primary checked:bg-primary checked:text-primary-content"
      type="radio"
      {name}
      value={option.value}
      aria-label={option.label}
      checked={value === option.value}
      onchange={() => (value = option.value)}
    />
  {/each}
</fieldset>
