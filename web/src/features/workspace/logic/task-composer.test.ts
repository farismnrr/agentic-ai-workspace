import { expect, it, vi } from 'vitest';
import { submitTaskFollowUp } from './task-composer';

it('keeps the draft and attached paths when an external chat send fails', async () => {
  let draft = 'Explain this file';
  let attachments = ['uploads/report.pdf'];
  const send = vi.fn().mockResolvedValue({ accepted: false });
  const clear = (): void => {
    draft = '';
    attachments = [];
  };

  const result = await submitTaskFollowUp(draft, attachments, true, send, clear);

  expect(result).toEqual({ accepted: false });
  expect(send).toHaveBeenCalledWith(
    'Explain this file\n\nAttached files (already in the project):\n- uploads/report.pdf — read it with pdftotext',
    true,
  );
  expect(draft).toBe('Explain this file');
  expect(attachments).toEqual(['uploads/report.pdf']);
});
