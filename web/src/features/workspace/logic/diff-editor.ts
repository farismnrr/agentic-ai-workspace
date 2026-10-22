import type { Extension } from '@codemirror/state';
import { EditorView } from '@codemirror/view';
import { languageFor } from './editor';

export interface DiffOptions {
  parent: HTMLElement;
  path: string;
  before: string;
  after: string;
}

export interface DiffEditor {
  destroy: () => void;
}

const collapse = { margin: 2, minSize: 4 };

const markerTheme = EditorView.theme({
  '.cm-merge-b .cm-changedLine::before': {
    content: '"+"', color: 'var(--color-success)', display: 'inline-block',
    fontWeight: '700', paddingRight: '0.5em', userSelect: 'none',
  },
  '.cm-deletedLine::before': {
    content: '"−"', color: 'var(--color-error)', display: 'inline-block',
    fontWeight: '700', paddingRight: '0.5em', userSelect: 'none',
  },
});

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
    theme.editorTheme, theme.syntaxColors, theme.diffTheme, markerTheme, await languageFor(options.path),
  ];

  const view = new EditorView({
    parent: options.parent, doc: options.after,
    extensions: [...common, merge.unifiedMergeView({ original: options.before, mergeControls: false, gutter: true, collapseUnchanged: collapse })],
  });
  return { destroy: () => { view.destroy(); } };
}
