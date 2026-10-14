<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';
  import type { Change } from '../../logic/task-types';

  interface Props {
    changes: Change[];
    onundo: (eventId: string) => void;
  }

  const { changes, onundo }: Props = $props();
</script>

{#if changes.length === 0}
  <p class="rounded-lg border border-dashed border-[var(--brand-border-strong)] p-4 text-[var(--brand-text-secondary)]">Nothing has been changed in this project yet.</p>
{:else}
  <ul class="flex flex-col gap-2">
    {#each changes as change (change.id)}
      <li class="rounded-lg border border-[var(--brand-border)] bg-base-100 p-3 text-sm">
        <p class="flex items-center gap-2 font-medium"><Icon name="pencil" class="size-4 text-secondary" />{change.text}</p>
        <ul class="mt-2 flex flex-wrap gap-1">
          {#each change.files as file (file)}<li><code class="rounded bg-base-200 px-1.5 py-0.5 text-xs">{file}</code></li>{/each}
        </ul>
        <div class="mt-2">
          {#if change.undone}<span class="badge badge-ghost badge-sm">Undone</span>
          {:else if change.canUndo}<button class="btn btn-outline btn-xs" type="button" onclick={() => { onundo(change.id); }}>Undo this change</button>{/if}
        </div>
      </li>
    {/each}
  </ul>
{/if}
