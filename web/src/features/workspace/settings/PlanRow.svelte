<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import { programs } from '../content/subscriptions';
  import type { Program } from '../logic/subscriptions-api';

  interface Props {
    program: Program;
    onremove: () => void;
  }

  const { program, onremove }: Props = $props();
  const copy = $derived(programs[program]);
</script>

<li class="flex flex-wrap items-center gap-3 py-3">
  <span class="grid size-8 shrink-0 place-items-center rounded-lg bg-base-300 text-sm font-medium" aria-hidden="true">{copy.name.slice(0, 1)}</span>
  <span class="flex min-w-0 flex-1 flex-col leading-tight">
    <span class="font-medium">{copy.name} <span class="font-normal text-xs text-[var(--brand-text-tertiary)]">· {copy.badge}</span></span>
    <span class="text-xs text-[var(--brand-text-secondary)]">{copy.connected}</span>
  </span>
  <span class="flex items-center gap-1 text-xs"><Icon name="check-circle" class="size-4 text-success" /> Signed in</span>
  <button class="btn btn-ghost btn-xs btn-square" type="button" aria-label="Sign out of {copy.name}" onclick={onremove}><Icon name="trash" class="size-4" /></button>
</li>
