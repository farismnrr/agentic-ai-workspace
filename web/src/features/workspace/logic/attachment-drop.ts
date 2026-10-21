const pastedImageExtensions: Record<string, string> = {
  'image/png': 'png',
  'image/jpeg': 'jpg',
  'image/webp': 'webp',
  'image/gif': 'gif',
};
let pastedImageNumber = 0;

export function hasFileDrag(types: readonly string[]): boolean {
  return Array.from(types).includes('Files');
}

export function droppedFiles(files: readonly File[]): File[] {
  return Array.from(files);
}

export function pastedImages(files: readonly File[]): File[] {
  return Array.from(files).flatMap((file) => {
    const extension = pastedImageExtensions[file.type];
    if (extension === undefined) return [];
    pastedImageNumber += 1;
    return [new File([file], `pasted-image-${String(pastedImageNumber)}.${extension}`, { type: file.type, lastModified: file.lastModified })];
  });
}

export function createAttachmentDropHandlers(
  onFiles: (files: File[]) => void,
  onActiveChange: (active: boolean) => void,
  enabled: () => boolean,
): {
  dragenter: (event: DragEvent) => void;
  dragover: (event: DragEvent) => void;
  dragleave: (event: DragEvent) => void;
  drop: (event: DragEvent) => void;
} {
  let depth = 0;
  return {
    dragenter(event) {
      const transfer = event.dataTransfer;
      if (transfer === null || !hasFileDrag(transfer.types)) return;
      event.preventDefault();
      if (!enabled()) return;
      depth += 1;
      onActiveChange(true);
    },
    dragover(event) {
      const transfer = event.dataTransfer;
      if (transfer === null || !hasFileDrag(transfer.types)) return;
      event.preventDefault();
      transfer.dropEffect = enabled() ? 'copy' : 'none';
    },
    dragleave(event) {
      const transfer = event.dataTransfer;
      if (!enabled() || transfer === null || !hasFileDrag(transfer.types)) return;
      depth = Math.max(0, depth - 1);
      if (depth === 0) onActiveChange(false);
    },
    drop(event) {
      const transfer = event.dataTransfer;
      if (transfer === null || !hasFileDrag(transfer.types)) return;
      event.preventDefault();
      depth = 0;
      onActiveChange(false);
      if (enabled()) onFiles(droppedFiles(Array.from(transfer.files)));
    },
  };
}
