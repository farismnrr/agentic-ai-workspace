<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';
  import type { Change } from '../../content/task-types';

  interface Props {
    changes: Change[];
  }

  const { changes }: Props = $props();
</script>

{#if changes.length === 0}
  <p class="rounded-lg border border-dashed border-[var(--brand-border-strong)] p-4 text-[var(--brand-text-secondary)]">Nothing has been changed on your site yet.</p>
{:else}
  <ul class="flex flex-col gap-2">
    {#each changes as change (change.id)}
      <li>
        <details class="collapse collapse-arrow rounded-lg border border-[var(--brand-border)] bg-base-100">
          <summary class="collapse-title flex min-h-10 items-center gap-2 text-sm font-medium"><Icon name="pencil" class="size-4 text-secondary" />{change.text}</summary>
          <div class="collapse-content text-sm">
            {#if change.files.length === 0}
              <p class="text-[var(--brand-text-secondary)]">No file details for this change.</p>
            {:else}
              <p class="mb-1 text-[var(--brand-text-secondary)]">Files touched</p>
              <ul class="flex flex-col gap-1">
                {#each change.files as file (file)}<li><code class="rounded bg-base-200 px-1.5 py-0.5 text-xs">{file}</code></li>{/each}
              </ul>
            {/if}
          </div>
        </details>
      </li>
    {/each}
  </ul>
{/if}
