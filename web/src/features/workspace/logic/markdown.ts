/** An item of a list; depth is how far it is indented (0 to 2). */
export interface ListItem {
  text: string;
  depth: number;
  checked?: boolean | undefined;
}

/** A document made of the markdown shapes agents write: headings, lists, tables, code, quotes and paragraphs. */
export type DocBlock =
  | { kind: 'heading'; level: number; text: string }
  | { kind: 'paragraph'; text: string }
  | { kind: 'list'; ordered: boolean; items: ListItem[] }
  | { kind: 'code'; text: string }
  | { kind: 'table'; head: string[]; rows: string[][] }
  | { kind: 'rule' }
  | { kind: 'quote'; text: string };

const heading = /^(#{1,4})\s+(.*)$/;
const bullet = /^[-*•]\s+(.*)$/;
const numbered = /^\d+[.)]\s+(.*)$/;
const rule = /^([-*_])(\s*\1){2,}$/;
const separator = /^\|?[\s:|-]*-[\s:|-]*\|?$/;
const maxDepth = 2;

function listItem(line: string): { ordered: boolean; text: string; checked?: boolean } | undefined {
  const dash = bullet.exec(line);
  if (dash?.[1] !== undefined) {
    const check = /^\[([ xX])\]\s+(.*)$/.exec(dash[1]);
    return check === null ? { ordered: false, text: dash[1] } : { ordered: false, text: check[2] ?? '', checked: check[1]?.toLowerCase() === 'x' };
  }
  const number = numbered.exec(line);
  return number?.[1] === undefined ? undefined : { ordered: true, text: number[1] };
}

const cells = (line: string): string[] => line.replace(/^\||\|$/g, '').split('|').map((cell) => cell.trim());

/** Adds a table row, as the head of a new table or a row of the table before it; the --- line is dropped. */
function addRow(blocks: DocBlock[], line: string): void {
  const last = blocks.at(-1);
  if (last?.kind === 'table') {
    if (!separator.test(line)) last.rows.push(cells(line));
  } else blocks.push({ kind: 'table', head: cells(line), rows: [] });
}

/** Adds a line to the blocks: a heading, a rule, a table row, an item of the list before it, a new list, or a paragraph. */
function add(blocks: DocBlock[], line: string, indent: number): void {
  const head = heading.exec(line);
  if (head?.[1] !== undefined && head[2] !== undefined) {
    blocks.push({ kind: 'heading', level: head[1].length, text: head[2] });
    return;
  }
  if (rule.test(line)) {
    blocks.push({ kind: 'rule' });
    return;
  }
  if (line.startsWith('|')) {
    addRow(blocks, line);
    return;
  }
  const item = listItem(line);
  const last = blocks.at(-1);
  const depth = Math.min(maxDepth, Math.floor(indent / 2));
  if (item === undefined) blocks.push({ kind: 'paragraph', text: line.replace(/^>\s?/, '') });
  else if (last?.kind === 'list' && last.ordered === item.ordered) last.items.push({ text: item.text, depth, checked: item.checked });
  else blocks.push({ kind: 'list', ordered: item.ordered, items: [{ text: item.text, depth, checked: item.checked }] });
}

/** Reads markdown text into blocks that can be shown as headings, lists, tables and code without any HTML. */
export function parseDoc(markdown: string): DocBlock[] {
  const blocks: DocBlock[] = [];
  let fence: string[] | undefined;
  for (const raw of markdown.split('\n')) {
    const line = raw.trim();
    if (fence !== undefined) {
      if (line.startsWith('```')) {
        blocks.push({ kind: 'code', text: fence.join('\n') });
        fence = undefined;
      } else fence.push(raw);
    } else if (line.startsWith('```')) fence = [];
    else if (line.startsWith('>')) blocks.push({ kind: 'quote', text: line.replace(/^>\s?/, '') });
    else if (line !== '') add(blocks, line, raw.length - raw.trimStart().length);
  }
  // An answer still being written can end inside a code block.
  if (fence !== undefined) blocks.push({ kind: 'code', text: fence.join('\n') });
  return blocks;
}
