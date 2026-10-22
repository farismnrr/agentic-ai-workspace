<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';
  import { rawUrl } from '../../logic/files-api';
  import type { SentFile } from '../../logic/message-attachments';

  interface Props {
    projectId: string;
    files: SentFile[];
  }

  const { projectId, files }: Props = $props();
</script>

<ul class="flex flex-wrap justify-end gap-2" aria-label="Attached files">
  {#each files as file (file.path)}
    <li class="max-w-full min-w-0">
      {#if file.photo}
        <a href={rawUrl(projectId, file.path)} target="_blank" rel="noopener" aria-label="Open {file.name}">
          <img class="max-h-48 max-w-full rounded-2xl border border-[var(--brand-border)] object-cover" src={rawUrl(projectId, file.path)} alt={file.name} loading="lazy" />
        </a>
      {:else}
        <a class="flex items-center gap-2 rounded-xl border border-[var(--brand-border)] bg-base-200 p-1.5 pr-3" href={rawUrl(projectId, file.path)} target="_blank" rel="noopener">
          <span class="grid size-10 shrink-0 place-items-center rounded-lg bg-base-100 text-[var(--brand-text-secondary)]"><Icon name="file-text" class="size-5" /></span>
          <span class="max-w-44 truncate text-xs font-medium">{file.name}</span>
        </a>
      {/if}
    </li>
  {/each}
</ul>
