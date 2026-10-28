import { defaultKeymap, history, historyKeymap } from '@codemirror/commands';
import { Compartment, EditorState } from '@codemirror/state';
import { EditorView, highlightActiveLine, keymap, lineNumbers } from '@codemirror/view';
import { unifiedMergeView } from '@codemirror/merge';
import type { Editor, EditorOptions } from './editor';
import { languageFor } from './editor';
import * as theme from './editor-theme';

export async function createEditor(options: EditorOptions): Promise<Editor> {
  let quiet = false;
  const gitBaseline = new Compartment();
  const view = new EditorView({
    parent: options.parent,
    state: EditorState.create({
      doc: options.doc,
      extensions: [
        lineNumbers(),
        highlightActiveLine(),
        history(),
        keymap.of([...defaultKeymap, ...historyKeymap]),
        theme.editorTheme,
        theme.syntaxColors,
        theme.diffTheme,
        await languageFor(options.path),
        gitBaseline.of(gitExtension(options.original)),
        EditorState.readOnly.of(options.readOnly),
        EditorView.editable.of(!options.readOnly),
        EditorView.updateListener.of((update) => {
          if (update.docChanged && !quiet) {
            options.onchange(update.state.doc.toString());
          }
        }),
      ],
    }),
  });

  return {
    text: () => view.state.doc.toString(),
    replace: (text) => {
      quiet = true;
      view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: text } });
      quiet = false;
    },
    setGitBaseline: (text) => { view.dispatch({ effects: gitBaseline.reconfigure(gitExtension(text)) }); },
    destroy: () => {
      view.destroy();
    },
  };
}

function gitExtension(original: string | undefined) {
  if (original === undefined) return [];
  return unifiedMergeView({
    original,
    gutter: true,
    highlightChanges: false,
    mergeControls: false,
    syntaxHighlightDeletions: false,
  });
}
