/** The partial workflow modifier directly before the cursor, if any. */
export function mentionQuery(text: string, caret: number): string | null {
  if (caret < 0) return null;
  const match = /(?:^|\s)@([\w-]*)$/.exec(text.slice(0, caret));
  return match?.[1] ?? null;
}

export function insertMention(text: string, caret: number, key: string): { text: string; caret: number } {
  const query = mentionQuery(text, caret);
  if (query === null) return { text, caret };
  const start = caret - query.length - 1;
  const before = text.slice(0, start);
  // The backend only accepts @deep-review on a standalone line.
  const after = text.slice(caret);
  const prefix = before.length > 0 && !before.endsWith('\n') ? '\n' : '';
  const suffix = after.length > 0 && !after.startsWith('\n') ? '\n' : '';
  const inserted = `${prefix}@${key}${suffix}`;
  return { text: before + inserted + after, caret: before.length + inserted.length };
}
