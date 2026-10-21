import { uploadAttachment, type Attachment } from './attachments';

interface QueuedUpload {
  projectId: string;
  attachment: Attachment;
  controller: AbortController;
}

interface UploadQueueOptions {
  onComplete: (id: string, attachment: Attachment) => void;
  onProjectChange: () => void;
}

/** Runs at most four uploads at once and ignores completions after cancellation. */
export function createAttachmentUploader(options: UploadQueueOptions): {
  reconcile: (projectId: string | undefined, attachments: Attachment[]) => void;
  cancel: (id: string) => void;
  dispose: () => void;
} {
  const active = new Map<string, AbortController>();
  const queue: QueuedUpload[] = [];
  let running = 0;
  let project = '';

  function pump(): void {
    while (running < 4 && queue.length > 0) {
      const upload = queue.shift();
      if (upload === undefined || upload.controller.signal.aborted) continue;
      running += 1;
      void uploadAttachment(upload.projectId, upload.attachment, upload.controller.signal)
        .then((result) => {
          if (!upload.controller.signal.aborted) options.onComplete(upload.attachment.id, result);
        })
        .finally(() => {
          running -= 1;
          if (active.get(upload.attachment.id) === upload.controller) active.delete(upload.attachment.id);
          pump();
        });
    }
  }

  function cancel(id: string): void {
    active.get(id)?.abort();
    active.delete(id);
    for (let index = queue.length - 1; index >= 0; index -= 1) {
      if (queue[index]?.attachment.id === id) queue.splice(index, 1);
    }
  }

  function reconcile(projectId: string | undefined, attachments: Attachment[]): void {
    if (projectId === undefined || projectId === '') return;
    if (project !== '' && project !== projectId) {
      for (const id of active.keys()) cancel(id);
      queue.length = 0;
      options.onProjectChange();
    }
    project = projectId;
    for (const attachment of attachments) {
      if (attachment.status !== 'loading' || active.has(attachment.id)) continue;
      const controller = new AbortController();
      active.set(attachment.id, controller);
      queue.push({ projectId, attachment, controller });
    }
    pump();
  }

  return {
    reconcile,
    cancel,
    dispose() {
      for (const id of active.keys()) cancel(id);
    },
  };
}
