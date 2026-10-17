<script lang="ts">
  interface Props {
    plan?: boolean | undefined;
    disabled?: boolean | undefined;
  }

  let { plan = $bindable(true), disabled = false }: Props = $props();
</script>

<label class="flex cursor-pointer items-center gap-1.5 rounded-full px-2 py-1 text-sm hover:bg-base-200 {plan ? 'font-medium' : 'text-[var(--brand-text-secondary)]'}" title="The team writes the plan and waits for your yes before it builds anything">
  <input class="toggle toggle-xs toggle-primary" type="checkbox" bind:checked={plan} {disabled} />Plan first
</label>
