import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  attachmentsCanSend,
  hasFailedUploads,
  hasPendingUploads,
  readyPaths,
  updateAttachment,
  uploadAttachment,
  validateFile,
  validateSelection,
  withAttachments,
  type Attachment,
} from './attachments';

function file(name: string, size = 1, type = ''): File {
  return new File([new Uint8Array(size)], name, { type });
}

function loading(id = 'one', name = 'one.pdf'): Attachment {
  return { id, file: file(name), status: 'loading' };
}

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe('attachment rules', () => {
  it('accepts supported extensions with matching, missing, or generic browser MIME', () => {
    expect(validateFile(file('photo.png', 2, 'image/png'))).toBeUndefined();
    expect(validateFile(file('notes.md'))).toBeUndefined();
    expect(validateFile(file('bundle.zip', 2, 'application/zip'))).toBeUndefined();
    expect(validateFile(file('archive.zip', 2, 'application/x-zip-compressed'))).toBeUndefined();
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
    expect(validateFile(file('archive.zip', 25 * 1024 * 1024 + 1))).toContain('25 MB');
    expect(validateFile(file('fake.zip', 10, 'application/pdf'))).toContain('does not match');
  });

  it('limits a message to ten accepted files', () => {
    const selection = validateSelection([file('one.txt'), file('two.txt')], 9);
    expect(selection.accepted).toHaveLength(1);
    expect(selection.errors[0]).toContain('up to 10');
  });
});

describe('attachment uploads', () => {
  it('moves a loading item to ready and exposes its path', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(Response.json({ path: 'uploads/one.pdf' })));
    const ready = await uploadAttachment('project', loading());
    expect(ready).toMatchObject({ id: 'one', status: 'ready', path: 'uploads/one.pdf' });
    expect(readyPaths([ready])).toEqual(['uploads/one.pdf']);
  });

  it('moves a loading item to failed and supports retrying only that item', async () => {
    const fetch = vi.fn()
      .mockResolvedValueOnce(Response.json({ error: 'unknown' }, { status: 500 }))
      .mockResolvedValueOnce(Response.json({ path: 'uploads/one.pdf' }));
    vi.stubGlobal('fetch', fetch);
    const failed = await uploadAttachment('project', loading());
    expect(failed.status).toBe('failed');
    if (failed.status !== 'failed') throw new Error('Expected failed upload');
    expect(failed.error).toContain('Upload failed');
    const ready = await uploadAttachment('project', { ...failed, status: 'loading' });
    expect(ready.status).toBe('ready');
    expect(fetch).toHaveBeenCalledTimes(2);
  });

  it('aborts a removed in-flight upload and ignores any late result by id', async () => {
    let finish: ((response: Response) => void) | undefined;
    vi.stubGlobal('fetch', vi.fn().mockImplementation((_input: RequestInfo | URL, init?: RequestInit) => new Promise<Response>((resolve, reject) => {
      finish = resolve;
      init?.signal?.addEventListener('abort', () => { reject(new DOMException('Aborted', 'AbortError')); });
    })));
    const controller = new AbortController();
    const pending = uploadAttachment('project', loading(), controller.signal);
    controller.abort();
    finish?.(Response.json({ path: 'uploads/one.pdf' }));
    await expect(pending).rejects.toMatchObject({ name: 'AbortError' });
    expect(updateAttachment([], 'one', (item) => item)).toEqual([]);
  });

  it('does not upload an already-ready item again', async () => {
    const fetch = vi.fn();
    vi.stubGlobal('fetch', fetch);
    const ready: Attachment = { ...loading(), status: 'ready', path: 'uploads/one.pdf' };
    await expect(uploadAttachment('project', ready)).resolves.toBe(ready);
    expect(fetch).not.toHaveBeenCalled();
  });
});

describe('attachment Send gating', () => {
  it('reports pending and failed states while text-only content remains unchanged', () => {
    expect(hasPendingUploads([loading()])).toBe(true);
    const failed: Attachment = { id: 'two', file: file('two.pdf'), status: 'failed', error: 'Try again.' };
    expect(hasFailedUploads([failed])).toBe(true);
    expect(withAttachments('Just text', [])).toBe('Just text');
  });

  it('allows Send only when every attachment is ready or removed', () => {
    const pending = loading();
    const ready: Attachment = { ...pending, status: 'ready', path: 'uploads/one.pdf' };
    const failed: Attachment = { ...pending, status: 'failed', error: 'Try again.' };

    expect(attachmentsCanSend([pending])).toBe(false);
    expect(attachmentsCanSend([failed])).toBe(false);
    expect(attachmentsCanSend([ready])).toBe(true);
    expect(attachmentsCanSend([pending].filter((item) => item.id !== pending.id))).toBe(true);
  });
});

describe('attachment prompts', () => {
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
