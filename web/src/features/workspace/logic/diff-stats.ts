export interface DiffStats {
  added: number;
  removed: number;
}

/** How many lines a change added and removed, counted by the same comparison that draws the diff. */
export async function diffStats(before: string, after: string): Promise<DiffStats> {
  const runtime = await import('./diff-stats-runtime');
  return runtime.diffStats(before, after);
}
