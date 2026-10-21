import { describe, expect, it } from 'vitest';
import { createAttachmentDropHandlers, droppedFiles, hasFileDrag, pastedImages } from './attachment-drop';

function file(name: string, type: string): File {
  return new File(['image'], name, { type });
}

describe('attachment drop and paste filtering', () => {
  it('recognizes file drags and ignores other drag types', () => {
    expect(hasFileDrag(['Files'])).toBe(true);
    expect(hasFileDrag(['text/plain'])).toBe(false);
  });

  it('keeps dropped files for the shared attachment validation path', () => {
    const files = [file('report.pdf', 'application/pdf'), file('notes.txt', 'text/plain')];
    expect(droppedFiles(files)).toEqual(files);
  });

  it('adds pasted images with predictable names and ignores non-images', () => {
    const images = pastedImages([
      file('clipboard', 'image/png'),
      file('photo', 'image/jpeg'),
      file('notes.txt', 'text/plain'),
    ]);
    expect(images.map((image) => [image.name, image.type])).toEqual([
      ['pasted-image-1.png', 'image/png'],
      ['pasted-image-2.jpg', 'image/jpeg'],
    ]);
  });

  it('prevents browser navigation for file drags while disabled without adding files', () => {
    let prevented = false;
    let added = false;
    const handlers = createAttachmentDropHandlers(() => { added = true; }, () => undefined, () => false);
    const event = {
      dataTransfer: { types: ['Files'], files: [file('clip.mp4', 'video/mp4')], dropEffect: 'none' },
      preventDefault: () => { prevented = true; },
    } as unknown as DragEvent;
    handlers.dragover(event);
    handlers.drop(event);
    expect(prevented).toBe(true);
    expect(added).toBe(false);
  });
});
