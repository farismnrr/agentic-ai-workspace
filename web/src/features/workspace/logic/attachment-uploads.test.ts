import { afterEach, describe, expect, it, vi } from 'vitest';
import { createAttachmentUploader } from './attachment-uploads';
import type { Attachment } from './attachments';

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe('attachment upload queue', () => {
  it('aborts removal and ignores a completion that arrives afterward', async () => {
    let finish: ((response: Response) => void) | undefined;
    let signal: AbortSignal | null | undefined;
    vi.stubGlobal('fetch', vi.fn().mockImplementation((_input: RequestInfo | URL, init?: RequestInit) => {
      signal = init?.signal;
      return new Promise<Response>((resolve) => { finish = resolve; });
    }));
    const onComplete = vi.fn();
    const attachment: Attachment = { id: 'item', file: new File(['x'], 'item.txt'), status: 'loading' };
    const uploader = createAttachmentUploader({ onComplete, onProjectChange: vi.fn() });

    uploader.reconcile('project', [attachment]);
    uploader.cancel(attachment.id);
    finish?.(Response.json({ path: 'uploads/item.txt' }));
    expect(signal?.aborted).toBe(true);
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(onComplete).not.toHaveBeenCalled();
    uploader.dispose();
  });
});
