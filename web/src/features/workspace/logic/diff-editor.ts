import type { Extension } from '@codemirror/state';
import { languageFor } from './editor';

export interface DiffOptions {
  parent: HTMLElement;
  path: string;
  before: string;
  after: string;
  /** Old and new side by side; otherwise one column with the removed lines among the added. */
  split: boolean;
}

export interface DiffEditor {
  destroy: () => void;
}

const collapse = { margin: 2, minSize: 4 };

/** Shows how a file changed. The comparison code is loaded only when a diff is opened. */
export async function createDiff(options: DiffOptions): Promise<DiffEditor> {
  const [{ EditorView }, { EditorState }, merge, theme] = await Promise.all([
    import('@codemirror/view'),
    import('@codemirror/state'),
    import('@codemirror/merge'),
    import('./editor-theme'),
  ]);
  const common: Extension[] = [
    EditorState.readOnly.of(true), EditorView.editable.of(false), EditorView.lineWrapping,
    theme.editorTheme, theme.syntaxColors, theme.diffTheme, await languageFor(options.path),
  ];

  if (options.split) {
    const view = new merge.MergeView({
      parent: options.parent, gutter: true, highlightChanges: true, collapseUnchanged: collapse,
      a: { doc: options.before, extensions: common }, b: { doc: options.after, extensions: common },
    });
    return { destroy: () => { view.destroy(); } };
  }

  const view = new EditorView({
    parent: options.parent, doc: options.after,
    extensions: [...common, merge.unifiedMergeView({ original: options.before, mergeControls: false, gutter: true, collapseUnchanged: collapse })],
  });
  return { destroy: () => { view.destroy(); } };
}
