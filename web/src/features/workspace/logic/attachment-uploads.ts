import { uploadAttachment, type Attachment } from './attachments';
import { discardZIP } from './zip-upload';

interface QueuedUpload {
  projectId: string;
  attachment: Attachment;
  controller: AbortController;
}
interface UploadQueueOptions {
  onComplete: (id: string, attachment: Attachment) => void;
  onProgress?: (id: string, bytes: number, phase: 'uploading' | 'validating') => void;
  onProjectChange: () => void;
}

/** Four ordinary files maximum; only one large ZIP per project at once.
 * Paused/canceled requests never update the chat after abort. */
export function createAttachmentUploader(options: UploadQueueOptions): {
  reconcile: (projectId: string | undefined, attachments: Attachment[]) => void;
  cancel: (id: string) => void;
  discard: (id: string, file: File) => void;
  dispose: () => void;
} {
  const active = new Map<string, AbortController>();
  const queue: QueuedUpload[] = [];
  let running = 0;
  let zipRunning = false;
  let project = '';

  function pump(): void {
    while (running < 4 && queue.length > 0) {
      const next = queue[0];
      if (!next) break;
      if (next.attachment.file.name.toLowerCase().endsWith('.zip') && zipRunning) break;
      const upload = queue.shift();
      if (!upload || upload.controller.signal.aborted) continue;
      const isZip = upload.attachment.file.name.toLowerCase().endsWith('.zip');
      if (isZip) zipRunning = true;
      running++;
      void uploadAttachment(upload.projectId, upload.attachment, upload.controller.signal,
        (bytes, phase) => {
          if (!upload.controller.signal.aborted) options.onProgress?.(upload.attachment.id, bytes, phase);
        })
        .then(result => {
          if (!upload.controller.signal.aborted) options.onComplete(upload.attachment.id, result);
        })
        .catch(() => undefined) // abort is a pause/cancel, not an unhandled rejection
        .finally(() => {
          if (isZip) zipRunning = false;
          running--;
          if (active.get(upload.attachment.id) === upload.controller) active.delete(upload.attachment.id);
          pump();
        });
    }
  }

  function cancel(id: string): void {
    active.get(id)?.abort();
    active.delete(id);
    for (let i = queue.length - 1; i >= 0; i--) {
      if (queue[i]?.attachment.id === id) queue.splice(i, 1);
    }
  }
  function discard(id: string, file: File): void {
    cancel(id);
    if (project && file.name.toLowerCase().endsWith('.zip')) {
      void discardZIP(project, file);
    }
  }
  function reconcile(projectId: string | undefined, attachments: Attachment[]): void {
    if (!projectId) return;
    if (project !== '' && project !== projectId) {
      for (const id of active.keys()) cancel(id);
      queue.length = 0;
      options.onProjectChange();
    }
    project = projectId;
    for (const attachment of attachments) {
      if (attachment.status !== 'loading' || active.has(attachment.id)) continue;
      if (queue.some(entry => entry.attachment.id === attachment.id)) continue;
      const controller = new AbortController();
      active.set(attachment.id, controller);
      queue.push({ projectId, attachment, controller });
    }
    pump();
  }
  return {
    reconcile, cancel, discard,
    dispose() { for (const id of active.keys()) cancel(id); },
  };
}
