<script lang="ts">
  import type { Snippet } from 'svelte';
  import Icon from '../../../shared/ui/Icon.svelte';
  import AttachmentChips from './AttachmentChips.svelte';
  import AttachmentNotices from './AttachmentNotices.svelte';
  import { totalFileSize, validateSelection, type Attachment } from '../logic/attachments';
  import type { Project } from '../logic/task-types';
  import { createAttachmentDropHandlers, pastedImages } from '../logic/attachment-drop';

  interface Props {
    id: string;
    placeholder: string;
    text?: string | undefined;
    projects?: Project[] | undefined;
    project?: string | undefined;
    busy?: boolean | undefined;
    disabled?: boolean | undefined;
    error?: string | undefined;
    attachments?: Attachment[] | undefined;
    uploadProgress?: string | undefined;
    attachable?: boolean | undefined;
    tools?: Snippet | undefined;
    onsend: (text: string, project: string) => void;
  }

  let {
    id,
    placeholder,
    text = $bindable(''),
    projects,
    project = $bindable(''),
    busy = false,
    disabled = false,
    error = '',
    attachments = $bindable([]),
    uploadProgress = '',
    attachable = false,
    tools,
    onsend,
  }: Props = $props();

  const acceptedTypes = [
    '.png', '.jpg', '.jpeg', '.webp', '.gif', '.svg', '.pdf', '.txt', '.md', '.csv', '.json', '.log',
    '.mp4', '.webm', '.mov', 'image/png', 'image/jpeg', 'image/webp', 'image/gif', 'image/svg+xml',
    'application/pdf', 'text/plain', 'text/markdown', 'text/csv', 'application/json', 'video/mp4',
    'video/webm', 'video/quicktime',
  ].join(',');
  let pickerErrors = $state<string[]>([]);
  let dropActive = $state(false);
  const needsProject = $derived(projects !== undefined && project === '');
  const documentThreshold = 12_000;
  const length = $derived(Array.from(text).length);
  const totalSize = $derived(totalFileSize(attachments));
  const failedFiles = $derived(attachments.filter((item) => item.uploadError !== undefined));
  const alertMessages = $derived([
    error,
    ...pickerErrors,
    ...failedFiles.map((item) => `${item.file.name}: ${item.uploadError ?? 'Upload failed. Press Send to try again.'}`),
  ].filter((message) => message !== ''));
  const statusMessage = $derived([
    uploadProgress,
    ...(length > documentThreshold ? [`Long text · ${length.toLocaleString()} characters. It will be saved as a project document when you send.`] : []),
  ].filter((message) => message !== '').join(' '));
  const canSend = $derived((text.trim() !== '' || attachments.length > 0) && !needsProject && !busy && !disabled);

  function addFiles(list: FileList | File[]): void {
    if (busy || disabled) return;
    pickerErrors = [];
    const selection = validateSelection(Array.from(list), attachments.length);
    attachments = [...attachments, ...selection.accepted.map((file) => ({ id: crypto.randomUUID(), file }))];
    pickerErrors = selection.errors;
  }

  function attach(event: Event): void {
    const input = event.currentTarget as HTMLInputElement;
    if (input.files !== null) addFiles(input.files);
    input.value = '';
  }

  function remove(idToRemove: string): void {
    const index = attachments.findIndex((item) => item.id === idToRemove);
    attachments = attachments.filter((item) => item.id !== idToRemove);
    requestAnimationFrame(() => {
      const next = attachments[index] ?? attachments[index - 1];
      if (next !== undefined) document.querySelector<HTMLButtonElement>(`[data-attachment-remove="${next.id}"]`)?.focus();
      else document.getElementById(id)?.focus();
    });
  }

  function send(): void {
    if (canSend) onsend(text, project);
  }

  function onsubmit(event: SubmitEvent): void {
    event.preventDefault();
    send();
  }

  function onkeydown(event: KeyboardEvent): void {
    if (event.key !== 'Enter' || event.shiftKey || event.isComposing) return;
    event.preventDefault();
    send();
  }

  const dropHandlers = createAttachmentDropHandlers(
    (files) => { if (attachable && !disabled && !busy) addFiles(files); },
    (active) => { dropActive = attachable && !disabled && !busy && active; },
    () => attachable && !disabled && !busy,
  );

  function paste(event: ClipboardEvent): void {
    if (!attachable || disabled || busy || event.clipboardData === null) return;
    const images = pastedImages(Array.from(event.clipboardData.files));
    if (images.length > 0) addFiles(images);
  }
</script>

<form {onsubmit}>
  <label class="sr-only" for={id}>{placeholder}</label>
  <div
    role="group"
    aria-label="Message composer"
    aria-busy={busy}
    class="relative rounded-3xl border bg-base-100 shadow-lg
      focus-within:border-[var(--brand-border-strong)] {dropActive
        ? 'border-primary outline-2 outline-offset-2 outline-primary'
        : 'border-[var(--brand-border)]'}"
    ondragenter={dropHandlers.dragenter}
    ondragover={dropHandlers.dragover}
    ondragleave={dropHandlers.dragleave}
    ondrop={dropHandlers.drop}
  >
    {#if dropActive}
      <div class="pointer-events-none absolute inset-0 z-10 grid place-items-center rounded-3xl bg-base-100/90 text-sm font-medium text-primary">
        Drop files here
      </div>
    {/if}
    <textarea
      {id}
      bind:value={text}
      {onkeydown}
      onpaste={paste}
      rows="2"
      {placeholder}
      disabled={disabled || busy}
      class="field-sizing-content max-h-52 min-h-14 w-full resize-none bg-transparent
        px-4 pb-1 pt-3.5 text-sm focus:outline-none"
    ></textarea>
    {#if attachments.length > 0}
      <AttachmentChips {attachments} {busy} onremove={remove} />
    {/if}
    <div class="flex flex-wrap items-center gap-1 px-2 pb-2">
      {#if attachable}
        <label class="btn btn-ghost btn-circle btn-sm has-[:focus-visible]:outline has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-primary" title="Attach files">
          <Icon name="paperclip" class="size-4" />
          <span class="sr-only">Attach files</span>
          <input
            type="file"
            class="sr-only"
            accept={acceptedTypes}
            multiple
            disabled={disabled || busy}
            onchange={attach}
          />
        </label>
      {/if}
      {#if projects !== undefined}
        <label class="sr-only" for="{id}-project">Project</label>
        <select
          id="{id}-project"
          class="select select-ghost select-sm w-auto rounded-full
            {needsProject ? 'text-[var(--brand-text-secondary)]' : 'font-medium'}"
          bind:value={project}
          disabled={disabled || busy}
        >
          <option value="" disabled>Select a project</option>
          {#each projects as item (item.id)}<option value={item.id}>{item.name}</option>{/each}
        </select>
      {/if}
      {#if tools !== undefined}{@render tools()}{/if}
      <button class="btn btn-primary btn-circle btn-sm ml-auto" type="submit" aria-label={busy ? 'Sending' : 'Send'} disabled={!canSend}>
        {#if busy}
          <span class="loading loading-spinner loading-xs"></span>
        {:else}
          <Icon name="arrow-right" class="size-4" />
        {/if}
      </button>
    </div>
  </div>
  {#if needsProject && (text.trim() !== '' || attachments.length > 0)}
    <p class="mt-2 text-sm text-[var(--brand-text-secondary)]">
      Select a project so I know where to work.
    </p>
  {/if}
  <AttachmentNotices warning={totalSize > 100 * 1024 * 1024} status={statusMessage} errors={alertMessages} />
</form>
