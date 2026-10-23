import { Text } from '@codemirror/state';
import { Chunk } from '@codemirror/merge';
import type { DiffStats } from './diff-stats';

export function diffStats(before: string, after: string): DiffStats {
  const a = Text.of(before.split('\n'));
  const b = Text.of(after.split('\n'));
  const lines = (doc: typeof a, from: number, end: number): number =>
    end > from ? doc.lineAt(end).number - doc.lineAt(from).number + 1 : 0;
  let added = 0;
  let removed = 0;
  for (const chunk of Chunk.build(a, b)) {
    removed += lines(a, chunk.fromA, chunk.endA);
    added += lines(b, chunk.fromB, chunk.endB);
  }
  return { added, removed };
}
