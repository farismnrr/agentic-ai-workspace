<script lang="ts">
  import { onDestroy } from 'svelte';
  import RoleMentionInput from './RoleMentionInput.svelte';
  import type { Snippet } from 'svelte';
  import Icon from '../../../shared/ui/Icon.svelte';
  import ComposerControls from './ComposerControls.svelte';
  import AttachmentChips from './AttachmentChips.svelte';
  import ComposerNotices from './ComposerNotices.svelte';
  import {
    attachmentsCanSend, totalFileSize, updateAttachment,
    validateSelection, type Attachment,
  } from '../logic/attachments';
  import type { Project } from '../logic/task-types';
  import { createAttachmentUploader } from '../logic/attachment-uploads';
  import { createAttachmentInputHandlers } from '../logic/attachment-drop';

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
    projectId?: string | undefined;
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
    projectId,
    uploadProgress = '',
    attachable = false,
    tools,
    onsend,
  }: Props = $props();

  let pickerErrors = $state<string[]>([]);
  let dropActive = $state(false);
  const uploader = createAttachmentUploader({
    onComplete: (id, result) => {
      attachments = updateAttachment(attachments, id, () => result);
    },
    onProjectChange: () => {
      attachments = attachments.map(({ id: itemId, file }) => ({ id: itemId, file, status: 'loading' }));
    },
  });
  const needsProject = $derived(projects !== undefined && project === '');
  const totalSize = $derived(totalFileSize(attachments));
  const canSend = $derived(
    (text.trim() !== '' || attachments.length > 0)
      && attachmentsCanSend(attachments) && !needsProject && !busy && !disabled,
  );

  $effect(() => {
    uploader.reconcile(projectId, attachments);
  });

  onDestroy(uploader.dispose);
  function addFiles(list: FileList | File[]): void {
    if (busy || disabled) return;
    pickerErrors = [];
    const selection = validateSelection(Array.from(list), attachments.length);
    attachments = [...attachments, ...selection.accepted.map((file) => ({
      id: crypto.randomUUID(), file, status: 'loading' as const,
    }))];
    pickerErrors = selection.errors;
  }

  function attach(event: Event): void {
    const input = event.currentTarget as HTMLInputElement;
    if (input.files !== null) addFiles(input.files);
    input.value = '';
  }

  function remove(idToRemove: string): void {
    uploader.cancel(idToRemove);
    const index = attachments.findIndex((item) => item.id === idToRemove);
    attachments = attachments.filter((item) => item.id !== idToRemove);
    requestAnimationFrame(() => {
      const next = attachments[index] ?? attachments[index - 1];
      if (next !== undefined) {
        document.querySelector<HTMLButtonElement>(`[data-attachment-remove="${next.id}"]`)?.focus();
      }
      else document.getElementById(id)?.focus();
    });
  }

  function retry(idToRetry: string): void {
    const item = attachments.find((attachment) => attachment.id === idToRetry);
    if (item?.status !== 'failed') return;
    attachments = updateAttachment(attachments, idToRetry, (attachment) => ({
      id: attachment.id, file: attachment.file, status: 'loading',
    }));
  }

  function send(): void { if (canSend) onsend(text, project); }

  function onsubmit(event: SubmitEvent): void {
    event.preventDefault();
    send();
  }

  function onkeydown(event: KeyboardEvent): void {
    if (event.key !== 'Enter' || event.shiftKey || event.isComposing) return;
    event.preventDefault();
    send();
  }

  const inputHandlers = createAttachmentInputHandlers(
    (files) => { if (attachable && !disabled && !busy) addFiles(files); },
    (active) => { dropActive = attachable && !disabled && !busy && active; },
    () => attachable && !disabled && !busy,
  );
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
    ondragenter={inputHandlers.drop.dragenter}
    ondragover={inputHandlers.drop.dragover}
    ondragleave={inputHandlers.drop.dragleave}
    ondrop={inputHandlers.drop.drop}
  >
    {#if dropActive}
      <div
        class="pointer-events-none absolute inset-0 z-10 grid place-items-center rounded-3xl
          bg-base-100/90 text-sm font-medium text-primary"
      >
        Drop files here
      </div>
    {/if}
    <RoleMentionInput {id} bind:text {placeholder} disabled={disabled || busy}
      {onkeydown} onpaste={inputHandlers.paste} />
    {#if attachments.length > 0}
      <AttachmentChips {attachments} {busy} waiting={needsProject} onremove={remove} onretry={retry} />
    {/if}
    <div class="flex flex-wrap items-center gap-1 px-2 pb-2">
      <ComposerControls {id} {projects} bind:project {attachable} {disabled} {busy} {tools} onattach={attach} />
      <button
        class="btn btn-primary btn-circle btn-sm ml-auto"
        type="submit"
        aria-label={busy ? 'Sending' : 'Send'}
        aria-describedby={!attachmentsCanSend(attachments) ? `${id}-attachment-reason` : undefined}
        disabled={!canSend}
      >
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
  <ComposerNotices
    {id}
    {attachments}
    {text}
    {uploadProgress}
    {error}
    pickerErrors={pickerErrors}
    warning={totalSize > 100 * 1024 * 1024}
  />
</form>
