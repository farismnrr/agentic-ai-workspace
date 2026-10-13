<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';

  interface Props {
    class?: string;
  }

  const { class: className = '' }: Props = $props();
</script>

<div class="card flex-row items-center gap-3 border border-[var(--brand-border)] bg-base-100 p-3 shadow-[var(--brand-shadow)] {className}" role="status">
  <span class="grid size-9 shrink-0 place-items-center rounded-lg bg-primary text-primary-content">
    <Icon name="check" class="size-5" />
  </span>
  <span class="text-sm leading-tight">
    Your task is finished
    <span class="block text-xs text-[var(--brand-text-tertiary)]">Just now</span>
  </span>
</div>
