<script lang="ts">
  import { onDestroy } from 'svelte';
  import { SvelteMap } from 'svelte/reactivity';
  import Icon from '../../../shared/ui/Icon.svelte';
  import {
    attachmentKind,
    formatFileSize,
    type Attachment,
  } from '../logic/attachments';

  interface Props {
    attachments: Attachment[];
    busy?: boolean;
    onremove: (id: string) => void;
  }

  let { attachments, busy = false, onremove }: Props = $props();
  const previews = new SvelteMap<File, string>();

  $effect(() => {
    const activeFiles = new Set(attachments.map((item) => item.file));
    for (const [file, url] of previews) {
      if (!activeFiles.has(file)) {
        URL.revokeObjectURL(url);
        previews.delete(file);
      }
    }
  });

  function previewUrl(file: File): string {
    let url = previews.get(file);
    if (url === undefined) {
      url = URL.createObjectURL(file);
      previews.set(file, url);
    }
    return url;
  }

  onDestroy(() => {
    for (const url of previews.values()) URL.revokeObjectURL(url);
    previews.clear();
  });
</script>

<ul class="flex flex-wrap gap-2 px-3 pb-2" aria-label="Attached files">
  {#each attachments as attachment (attachment.id)}
    <li
      class="relative flex max-w-full min-w-0 items-center gap-2 rounded-xl border
        {attachment.uploadError !== undefined ? 'border-error' : 'border-[var(--brand-border)]'} bg-base-200 p-1.5"
    >
      {#if attachmentKind(attachment.file) === 'photo'}
        <img
          class="size-10 rounded-lg object-cover"
          src={previewUrl(attachment.file)}
          alt=""
        />
      {:else}
        <span class="grid size-10 shrink-0 place-items-center rounded-lg bg-base-100 text-[var(--brand-text-secondary)]">
          <Icon name={attachmentKind(attachment.file) === 'video' ? 'video' : 'file-text'} class="size-5" />
        </span>
      {/if}
      <span class="min-w-0 flex-1">
        <span class="block max-w-44 truncate text-xs font-medium">{attachment.file.name}</span>
        <span class="block text-xs text-[var(--brand-text-secondary)] [overflow-wrap:anywhere]">
          {formatFileSize(attachment.file.size)}
          {#if attachment.path !== undefined} · Uploaded{/if}
          {#if attachment.uploadError !== undefined}<span class="text-error"> · Upload failed</span>{/if}
          {#if attachmentKind(attachment.file) === 'video'} · The AI can't watch videos.{/if}
        </span>
      </span>
      {#if attachment.uploadError !== undefined}<Icon name="alert-triangle" class="size-4 shrink-0 text-error" />{/if}
      <button
        type="button"
        class="btn btn-ghost btn-circle btn-xs"
        disabled={busy}
        aria-label="Remove {attachment.file.name}"
        data-attachment-remove={attachment.id}
        onclick={() => { if (!busy) onremove(attachment.id); }}
      >
        <Icon name="x" class="size-3" />
      </button>
    </li>
  {/each}
</ul>
