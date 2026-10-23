export interface DiffOptions {
  parent: HTMLElement;
  path: string;
  before: string;
  after: string;
}

export interface DiffEditor {
  destroy: () => void;
}

/** Shows how a file changed. The comparison code is loaded only when a diff is opened. */
export async function createDiff(options: DiffOptions): Promise<DiffEditor> {
  const runtime = await import('./diff-editor-runtime');
  return runtime.createDiff(options);
}
