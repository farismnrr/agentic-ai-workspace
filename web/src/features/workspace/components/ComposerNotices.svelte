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
  }

  const { id, attachments, text, uploadProgress, error, pickerErrors, warning }: Props = $props();
  const failedFiles = $derived(attachments.filter((item) => item.status === 'failed'));
  const errors = $derived([
    error,
    ...pickerErrors,
    ...failedFiles.map((item) => `${item.file.name}: ${item.error}`),
  ].filter((message) => message !== ''));
  const status = $derived(composerStatus(text, uploadProgress));
  const reason = $derived(
    hasPendingUploads(attachments)
      ? 'Wait for attachments to finish uploading before sending.'
      : hasFailedUploads(attachments)
        ? 'Retry or remove failed attachments before sending.'
        : '',
  );
</script>

{#if reason !== ''}
  <p id="{id}-attachment-reason" class="sr-only">{reason}</p>
{/if}
<AttachmentNotices {warning} {status} {errors} />
