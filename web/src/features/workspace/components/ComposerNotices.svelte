<script lang="ts">
  import AttachmentNotices from './AttachmentNotices.svelte';
  import {
    composerStatus,
    hasFailedUploads,
    hasPendingUploads,
    type Attachment,
  } from '../logic/attachments';

  interface Props {
    id: string;
    attachments: Attachment[];
    text: string;
    uploadProgress: string;
    error: string;
    pickerErrors: string[];
    warning: boolean;
    waiting: boolean;
  }

  const { id, attachments, text, uploadProgress, error, pickerErrors, warning, waiting }: Props = $props();
  const pendingCount = $derived(attachments.filter((item) => item.status === 'loading').length);
  const failedFiles = $derived(attachments.filter((item) => item.status === 'failed'));
  const errors = $derived([
    error,
    ...pickerErrors,
    ...failedFiles.map((item) => `${item.file.name}: ${item.error}`),
  ].filter((message) => message !== ''));
  const attachmentMessage = $derived(
    pendingCount > 0 && waiting
      ? 'Choose a project and your files will upload.'
      : pendingCount > 0
      ? `Uploading ${String(pendingCount)} attachment${pendingCount === 1 ? '' : 's'}.`
      : hasFailedUploads(attachments)
        ? 'An attachment failed to upload.'
        : attachments.length > 0 ? 'All attachments are ready.' : '',
  );
  const status = $derived(composerStatus(text, [uploadProgress, attachmentMessage].filter(Boolean).join(' ')));
  const reason = $derived(
    hasPendingUploads(attachments)
      ? waiting
        ? 'Choose a project so the attachments can upload.'
        : 'Wait for attachments to finish uploading before sending.'
      : hasFailedUploads(attachments)
        ? 'Retry or remove failed attachments before sending.'
        : '',
  );
</script>

{#if reason !== ''}
  <p id="{id}-attachment-reason" class="mt-2 text-sm text-base-content/70">{reason}</p>
{/if}
<AttachmentNotices {warning} {status} {errors} />
