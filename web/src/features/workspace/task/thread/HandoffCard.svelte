<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';
  import type { HandoffState } from '../../logic/task-types';

  interface Props {
    text: string;
    /** What the lead asked for, or what the teammate reported. */
    detail?: string | undefined;
    state: HandoffState;
  }

  const { text, detail, state }: Props = $props();
  const icon = $derived(state === 'assigned' ? 'arrow-right' : state === 'reported' ? 'check-circle' : 'x-circle');
  const tone = $derived(state === 'reported' ? 'text-success' : state === 'failed' ? 'text-error' : 'text-[var(--brand-text-tertiary)]');
</script>

<details class="group rounded-lg border border-[var(--brand-border)] bg-base-100 px-3 py-2">
  <summary class="flex cursor-pointer list-none items-center gap-2">
    <Icon name={icon} class="size-4 shrink-0 {tone}" />
    <span class="min-w-0 flex-1 truncate font-medium">{text}</span>
    {#if detail !== undefined && detail !== ''}<Icon name="chevron-down" class="size-3.5 shrink-0 transition-transform group-open:rotate-180" />{/if}
  </summary>
  {#if detail !== undefined && detail !== ''}<p class="mt-2 whitespace-pre-wrap break-words rounded-md bg-base-200 px-3 py-2 text-sm">{detail}</p>{/if}
</details>
