<script lang="ts">
  import { onDestroy } from 'svelte';
  import { SvelteMap } from 'svelte/reactivity';
  import Icon from '../../../shared/ui/Icon.svelte';
  import { attachmentKind, formatFileSize, type Attachment } from '../logic/attachments';

  interface Props {
    attachments: Attachment[];
    busy?: boolean;
    onremove: (id: string) => void;
    onretry: (id: string) => void;
  }

  let { attachments, busy = false, onremove, onretry }: Props = $props();
  const previews = new SvelteMap<File, string>();
  let selected = $state<Attachment | undefined>();
  let dialog = $state<HTMLDialogElement>();

  $effect(() => {
    const activeFiles = new Set(attachments.map((item) => item.file));
    for (const [file, url] of previews) {
      if (!activeFiles.has(file)) {
        URL.revokeObjectURL(url);
        previews.delete(file);
      }
    }
    if (selected !== undefined) {
      const selectedId = selected.id;
      const current = attachments.find((item) => item.id === selectedId);
      selected = current?.status === 'ready' ? current : undefined;
    }
  });

  $effect(() => {
    if (selected?.status === 'ready' && dialog?.open !== true) dialog?.showModal();
    if (selected === undefined && dialog?.open === true) dialog.close();
  });

  function previewUrl(file: File): string {
    let url = previews.get(file);
    if (url === undefined) {
      url = URL.createObjectURL(file);
      previews.set(file, url);
    }
    return url;
  }

  function onclose(): void {
    selected = undefined;
  }

  onDestroy(() => {
    for (const url of previews.values()) URL.revokeObjectURL(url);
    previews.clear();
  });
</script>

<ul class="flex flex-wrap gap-2 px-3 pb-2" aria-label="Attached files">
  {#each attachments as attachment (attachment.id)}
    {@const kind = attachmentKind(attachment.file)}
    <li
      class="relative flex max-w-full min-w-0 items-center gap-2 rounded-xl border
        {attachment.status === 'failed' ? 'border-error' : 'border-[var(--brand-border)]'} bg-base-200 p-1.5"
    >
      {#if kind === 'photo' && attachment.status === 'ready'}
        <button
          type="button"
          class="btn btn-ghost btn-square btn-sm p-0"
          aria-label="Preview {attachment.file.name}"
          onclick={() => (selected = attachment)}
        >
          <img class="size-10 rounded-lg object-cover" src={previewUrl(attachment.file)} alt="" />
        </button>
      {:else}
        <span class="grid size-10 shrink-0 place-items-center rounded-lg bg-base-100 text-[var(--brand-text-secondary)]">
          <Icon name={kind === 'video' ? 'video' : 'file-text'} class="size-5" />
        </span>
      {/if}
      <span class="min-w-0 flex-1">
        <span class="block max-w-44 truncate text-xs font-medium">{attachment.file.name}</span>
        <span class="block text-xs text-[var(--brand-text-secondary)] [overflow-wrap:anywhere]">
          {formatFileSize(attachment.file.size)}
          {#if attachment.status === 'ready'} · Uploaded{/if}
          {#if attachment.status === 'failed'}<span class="text-error"> · {attachment.error}</span>{/if}
          {#if kind === 'video'} · The AI can't watch videos.{/if}
        </span>
      </span>
      {#if attachment.status === 'loading'}
        <span class="loading loading-spinner loading-xs" aria-hidden="true"></span>
        <span class="text-xs" role="status">Uploading…</span>
      {/if}
      {#if attachment.status === 'failed'}
        <Icon name="alert-triangle" class="size-4 shrink-0 text-error" />
        <button
          type="button"
          class="btn btn-ghost btn-xs"
          disabled={busy}
          onclick={() => { onretry(attachment.id); }}
        >
          Retry
        </button>
      {/if}
      <button
        type="button"
        class="btn btn-ghost btn-circle btn-xs"
        aria-label="Remove {attachment.file.name}"
        data-attachment-remove={attachment.id}
        onclick={() => { onremove(attachment.id); }}
      >
        <Icon name="x" class="size-3" />
      </button>
    </li>
  {/each}
</ul>

<dialog bind:this={dialog} class="modal" onclose={onclose} aria-label="Image preview">
  {#if selected?.status === 'ready'}
    <div class="modal-box max-w-5xl p-3">
      <div class="mb-2 flex items-center justify-between gap-3">
        <p class="truncate text-sm font-medium">{selected.file.name}</p>
        <button type="button" class="btn btn-ghost btn-sm" onclick={() => dialog?.close()}>Close</button>
      </div>
      <img class="mx-auto max-h-[75dvh] max-w-full object-contain" src={previewUrl(selected.file)} alt={selected.file.name} />
    </div>
    <form method="dialog" class="modal-backdrop"><button aria-label="Close image preview">close</button></form>
  {/if}
</dialog>
