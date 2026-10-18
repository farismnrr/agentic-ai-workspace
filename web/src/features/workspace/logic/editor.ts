import type { EditorView } from '@codemirror/view';
import type { Extension } from '@codemirror/state';

/** What the page holds of one editor: its text, and the way to stop it. */
export interface Editor {
  text: () => string;
  /** Replaces the whole text without counting it as the person's edit. */
  replace: (text: string) => void;
  destroy: () => void;
}

export interface EditorOptions {
  parent: HTMLElement;
  path: string;
  doc: string;
  readOnly: boolean;
  onchange: (text: string) => void;
}

const javascript = async (typescript: boolean): Promise<Extension> => (await import('@codemirror/lang-javascript')).javascript({ jsx: true, typescript });

/** Each language is loaded only when a file of that kind is opened. */
const languages: Record<string, () => Promise<Extension>> = {
  js: () => javascript(false), mjs: () => javascript(false), cjs: () => javascript(false), jsx: () => javascript(false),
  ts: () => javascript(true), tsx: () => javascript(true),
  html: async () => (await import('@codemirror/lang-html')).html(),
  htm: async () => (await import('@codemirror/lang-html')).html(),
  svelte: async () => (await import('@codemirror/lang-html')).html(),
  vue: async () => (await import('@codemirror/lang-html')).html(),
  css: async () => (await import('@codemirror/lang-css')).css(),
  json: async () => (await import('@codemirror/lang-json')).json(),
  md: async () => (await import('@codemirror/lang-markdown')).markdown(),
  markdown: async () => (await import('@codemirror/lang-markdown')).markdown(),
  py: async () => (await import('@codemirror/lang-python')).python(),
  go: async () => (await import('@codemirror/lang-go')).go(),
};

/** The language of a file by its name, or none, so plain text still opens. */
export const languageFor = async (path: string): Promise<Extension> => (await languages[path.toLowerCase().split('.').at(-1) ?? '']?.()) ?? [];

/** Opens a code editor in the element. CodeMirror is loaded on first use, so the rest of the app does not carry it. */
export async function createEditor(options: EditorOptions): Promise<Editor> {
  const [{ EditorView, keymap, lineNumbers, highlightActiveLine }, { EditorState }, { defaultKeymap, history, historyKeymap }, theme] = await Promise.all([
    import('@codemirror/view'),
    import('@codemirror/state'),
    import('@codemirror/commands'),
    import('./editor-theme'),
  ]);
  let quiet = false;
  const view: EditorView = new EditorView({
    parent: options.parent,
    state: EditorState.create({
      doc: options.doc,
      extensions: [
        lineNumbers(), highlightActiveLine(), history(), keymap.of([...defaultKeymap, ...historyKeymap]),
        theme.editorTheme, theme.syntaxColors, await languageFor(options.path),
        EditorState.readOnly.of(options.readOnly), EditorView.editable.of(!options.readOnly),
        EditorView.updateListener.of((update) => { if (update.docChanged && !quiet) options.onchange(update.state.doc.toString()); }),
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
    destroy: () => { view.destroy(); },
  };
}
