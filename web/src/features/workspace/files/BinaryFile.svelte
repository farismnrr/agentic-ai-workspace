<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import { rawUrl } from '../logic/files-api';

  interface Props {
    projectId: string;
    path: string;
  }

  const { projectId, path }: Props = $props();
  const extension = $derived(path.slice(path.lastIndexOf('.') + 1).toLowerCase());
  const kind = $derived(extension === 'pdf' ? 'pdf' : ['png', 'jpg', 'jpeg', 'gif', 'webp'].includes(extension) ? 'picture' : 'other');
  const name = $derived(path.slice(path.lastIndexOf('/') + 1));
</script>

<div class="flex min-h-0 flex-1 flex-col gap-2 p-3">
  <div class="flex flex-wrap items-center gap-2">
    <a class="btn btn-primary btn-sm rounded-lg" href={rawUrl(projectId, path, true)} download={name}><Icon name="download" class="size-4" />Download</a>
    {#if kind === 'other'}<span class="text-sm text-[var(--brand-text-secondary)]">This file cannot be shown here. Download it to open it on your computer.</span>{/if}
  </div>
  {#if kind === 'pdf'}
    <iframe class="min-h-0 w-full flex-1 rounded-lg border border-[var(--brand-border)] bg-base-100" title={`Preview of ${name}`} src={rawUrl(projectId, path)}></iframe>
  {:else if kind === 'picture'}
    <div class="min-h-0 flex-1 overflow-auto"><img class="max-w-full rounded-lg border border-[var(--brand-border)]" alt={`Preview of ${name}`} src={rawUrl(projectId, path)} /></div>
  {/if}
</div>
