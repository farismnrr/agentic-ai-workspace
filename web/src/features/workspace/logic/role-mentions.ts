/** The partial role mention directly before the cursor, if any. */
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
  const inserted = `@${key} `;
  return { text: before + inserted + text.slice(caret), caret: before.length + inserted.length };
}
