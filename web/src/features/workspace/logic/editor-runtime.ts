import { defaultKeymap, history, historyKeymap } from '@codemirror/commands';
import { EditorState } from '@codemirror/state';
import { EditorView, highlightActiveLine, keymap, lineNumbers } from '@codemirror/view';
import type { Editor, EditorOptions } from './editor';
import { languageFor } from './editor';
import * as theme from './editor-theme';

export async function createEditor(options: EditorOptions): Promise<Editor> {
  let quiet = false;
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
        await languageFor(options.path),
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
    destroy: () => {
      view.destroy();
    },
  };
}
