import { describe, expect, it } from 'vitest';
import { withAttachments } from './attachments';
import { splitAttachments } from './message-attachments';

describe('splitAttachments', () => {
  it('returns what was written and the attached files', () => {
    const sent = withAttachments('Look at this', ['uploads/a.png', 'uploads/report.pdf']);
    const { text, files } = splitAttachments(sent);
    expect(text).toBe('Look at this');
    expect(files).toEqual([
      { path: 'uploads/a.png', name: 'a.png', photo: true },
      { path: 'uploads/report.pdf', name: 'report.pdf', photo: false },
    ]);
  });

  it('hides the filler sentence when only files were attached', () => {
    expect(splitAttachments(withAttachments('', ['uploads/a.png'])).text).toBe('');
  });

  it('leaves a message without files untouched', () => {
    expect(splitAttachments('hello')).toEqual({ text: 'hello', files: [] });
  });
});
