/** A piece of a line: plain text, emphasis, code or a link. Shown as elements, never as HTML. */
export type Span = { kind: 'text' | 'bold' | 'italic' | 'code'; text: string } | { kind: 'link'; text: string; href: string };

// Order matters: code first, so asterisks inside it stay as they are; then bold before italic; then links.
const token = /`([^`]+)`|\*\*([^*]+)\*\*|__([^_]+)__|\*([^*\s][^*]*)\*|\[([^\]]+)\]\((https?:\/\/[^\s)]+)\)|(https?:\/\/[^\s<>)]*[^\s<>).,;:!?])/g;

function emphasis(match: RegExpExecArray): Span | undefined {
  const [, code, bold, underscored, italic] = match;
  if (code !== undefined) return { kind: 'code', text: code };
  if (bold !== undefined || underscored !== undefined) return { kind: 'bold', text: bold ?? underscored ?? '' };
  return italic === undefined ? undefined : { kind: 'italic', text: italic };
}

function span(match: RegExpExecArray): Span {
  const [, , , , , label, target, bare] = match;
  if (label !== undefined && target !== undefined) return { kind: 'link', text: label, href: target };
  return emphasis(match) ?? { kind: 'link', text: bare ?? '', href: bare ?? '' };
}

/** Reads `code`, **bold**, *italic*, [links](https://...) and bare http(s) addresses. Other schemes stay text. */
export function parseInline(line: string): Span[] {
  const spans: Span[] = [];
  let from = 0;
  for (const match of line.matchAll(token)) {
    if (match.index > from) spans.push({ kind: 'text', text: line.slice(from, match.index) });
    spans.push(span(match));
    from = match.index + match[0].length;
  }
  if (from < line.length) spans.push({ kind: 'text', text: line.slice(from) });
  return spans;
}
