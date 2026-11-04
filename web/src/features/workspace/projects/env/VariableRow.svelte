<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';
  import type { EnvVariable } from './env-api';
  import { formatUpdatedAt, variableValue } from './env-logic';

  interface Props {
    item: EnvVariable;
    managedName: string;
    revealed?: { name: string; value: string } | undefined;
    onreveal: (item: EnvVariable) => void;
    onreplace: (item: EnvVariable) => void;
    ondelete: (item: EnvVariable) => void;
  }

  const { item, managedName, revealed, onreveal, onreplace, ondelete }: Props = $props();
</script>

<li class="flex flex-wrap items-center gap-3 py-3">
  <div class="min-w-0 flex-1">
    <p class="font-mono text-sm font-medium">{item.name}</p>
    <p class="break-all text-sm text-[var(--brand-text-secondary)]">
      {#if revealed?.name === item.name}{revealed.value}{:else}{variableValue(item)}{/if}
    </p>
    {#if item.name === managedName}<p class="mt-1 text-xs text-[var(--brand-text-secondary)]">Managed in the Database section.</p>{/if}
  </div>
  <div class="flex items-center gap-2 text-xs text-[var(--brand-text-secondary)]">
    {#if item.isSecret}<span class="badge badge-outline badge-sm">Secret</span>{/if}
    <time datetime={item.updatedAt}>{formatUpdatedAt(item.updatedAt)}</time>
  </div>
  {#if item.name !== managedName}
    {#if item.isSecret}
      <button class="btn btn-ghost btn-sm" type="button" aria-label="Reveal secret {item.name}" onclick={() => onreveal(item)}>
        <Icon name="eye" class="size-4" />Reveal
      </button>
    {/if}
    <button class="btn btn-ghost btn-sm" type="button" aria-label="Replace {item.name}" onclick={() => onreplace(item)}>
      <Icon name="pencil" class="size-4" />Replace
    </button>
    <button class="btn btn-ghost btn-sm text-error" type="button" aria-label="Delete {item.name}" onclick={() => ondelete(item)}>
      <Icon name="trash" class="size-4" />Delete
    </button>
  {/if}
</li>
