import { EditorState } from '@codemirror/state';
import { EditorView } from '@codemirror/view';
import * as merge from '@codemirror/merge';
import type { DiffEditor, DiffOptions } from './diff-editor';
import { languageFor } from './editor';
import * as theme from './editor-theme';

const collapse = { margin: 2, minSize: 4 };

const markerTheme = EditorView.theme({
  '.cm-merge-b .cm-changedLine::before': {
    content: '"+"',
    color: 'var(--color-success)',
    display: 'inline-block',
    fontWeight: '700',
    paddingRight: '0.5em',
    userSelect: 'none',
  },
  '.cm-deletedLine::before': {
    content: '"−"',
    color: 'var(--color-error)',
    display: 'inline-block',
    fontWeight: '700',
    paddingRight: '0.5em',
    userSelect: 'none',
  },
});

export async function createDiff(options: DiffOptions): Promise<DiffEditor> {
  const common = [
    EditorState.readOnly.of(true),
    EditorView.editable.of(false),
    EditorView.lineWrapping,
    theme.editorTheme,
    theme.syntaxColors,
    theme.diffTheme,
    markerTheme,
    await languageFor(options.path),
  ];
  const view = new EditorView({
    parent: options.parent,
    doc: options.after,
    extensions: [
      ...common,
      merge.unifiedMergeView({
        original: options.before,
        mergeControls: false,
        gutter: true,
        collapseUnchanged: collapse,
      }),
    ],
  });

  return {
    destroy: () => {
      view.destroy();
    },
  };
}
