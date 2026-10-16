/** A document made of the few markdown shapes a PRD uses: headings, bullets, numbered steps and paragraphs. */
export type DocBlock = { kind: 'heading'; level: number; text: string } | { kind: 'paragraph'; text: string } | { kind: 'list'; ordered: boolean; items: string[] };

const heading = /^(#{1,4})\s+(.*)$/;
const bullet = /^[-*•]\s+(.*)$/;
const numbered = /^\d+[.)]\s+(.*)$/;

function listItem(line: string): { ordered: boolean; text: string } | undefined {
  const dash = bullet.exec(line);
  if (dash?.[1] !== undefined) return { ordered: false, text: dash[1] };
  const number = numbered.exec(line);
  return number?.[1] === undefined ? undefined : { ordered: true, text: number[1] };
}

/** Adds a line to the blocks: a heading, an item of the list before it, a new list, or a paragraph. */
function add(blocks: DocBlock[], line: string): void {
  const head = heading.exec(line);
  if (head?.[1] !== undefined && head[2] !== undefined) {
    blocks.push({ kind: 'heading', level: head[1].length, text: head[2] });
    return;
  }
  const item = listItem(line);
  const last = blocks.at(-1);
  if (item === undefined) blocks.push({ kind: 'paragraph', text: line });
  else if (last?.kind === 'list' && last.ordered === item.ordered) last.items.push(item.text);
  else blocks.push({ kind: 'list', ordered: item.ordered, items: [item.text] });
}

/** Reads markdown text into blocks that can be shown as headings and lists without any HTML. */
export function parseDoc(markdown: string): DocBlock[] {
  const blocks: DocBlock[] = [];
  for (const raw of markdown.split('\n')) {
    const line = raw.trim();
    if (line !== '') add(blocks, line);
  }
  return blocks;
}
