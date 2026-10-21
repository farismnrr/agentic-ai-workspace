import { describe, expect, it } from 'vitest';
import {
  mergeUploadResult,
  validateFile,
  validateSelection,
  withAttachments,
  type Attachment,
} from './attachments';

function file(name: string, size = 1, type = ''): File {
  return new File([new Uint8Array(size)], name, { type });
}

describe('attachment rules', () => {
  it('accepts supported extensions with matching, missing, or generic browser MIME', () => {
    expect(validateFile(file('photo.png', 2, 'image/png'))).toBeUndefined();
    expect(validateFile(file('notes.md'))).toBeUndefined();
    expect(validateFile(file('clip.mov', 2, 'application/octet-stream'))).toBeUndefined();
  });

  it('rejects unsupported and mismatched types and empty files', () => {
    expect(validateFile(file('program.exe'))).toContain('not supported');
    expect(validateFile(file('picture.png', 2, 'application/pdf'))).toContain('does not match');
    expect(validateFile(file('empty.pdf', 0, 'application/pdf'))).toContain('empty');
  });

  it('uses the right per-type size limits', () => {
    expect(validateFile(file('image.svg', 5 * 1024 * 1024 + 1))).toContain('5 MB');
    expect(validateFile(file('clip.mp4', 100 * 1024 * 1024))).toBeUndefined();
    expect(validateFile(file('clip.webm', 100 * 1024 * 1024 + 1))).toContain('100 MB');
    expect(validateFile(file('book.pdf', 25 * 1024 * 1024 + 1))).toContain('25 MB');
  });

  it('limits a message to ten accepted files', () => {
    const selection = validateSelection([file('one.txt'), file('two.txt')], 9);
    expect(selection.accepted).toHaveLength(1);
    expect(selection.errors[0]).toContain('up to 10');
  });

  it('merges upload results without losing other file states', () => {
    const attachments: Attachment[] = [
      { id: 'first', file: file('first.pdf'), path: 'uploads/first.pdf' },
      { id: 'second', file: file('second.txt'), uploadError: 'Try again.' },
    ];
    const next = mergeUploadResult(attachments, 1, { path: 'uploads/second.txt' });
    expect(next.map((item) => item.path)).toEqual(['uploads/first.pdf', 'uploads/second.txt']);
    expect(next[1]?.uploadError).toBeUndefined();
    expect(next[1]?.id).toBe('second');
  });

  it('ignores an upload result for an attachment that is no longer present', () => {
    expect(mergeUploadResult([], 0, { path: 'uploads/removed.pdf' })).toEqual([]);
  });

  it('uses a useful prompt when the message contains only attachments', () => {
    expect(withAttachments('', ['uploads/report.pdf'])).toBe(
      'See the attached files.\n\nAttached files (already in the project):\n- uploads/report.pdf — read it with pdftotext',
    );
    expect(withAttachments('Please read this', [])).toBe('Please read this');
  });

  it('adds a clear reading hint for each supported attachment kind', () => {
    expect(withAttachments('Check these', [
      'uploads/photo.jpg', 'uploads/notes.md', 'uploads/clip.mp4',
    ])).toContain('view it with view_image (PNG only; convert other formats to PNG first)');
    expect(withAttachments('Check these', ['uploads/notes.md'])).toContain('read it directly');
    expect(withAttachments('Check these', ['uploads/clip.mp4'])).toContain(
      'stored only; you cannot watch video — tell the user what you can and cannot do',
    );
  });
});
