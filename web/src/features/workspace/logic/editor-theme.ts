import { EditorView } from '@codemirror/view';
import { HighlightStyle, syntaxHighlighting } from '@codemirror/language';
import { tags } from '@lezer/highlight';

// Colours come from the app's theme roles (daisyUI variables), never from a palette of the editor's own.
export const editorTheme = EditorView.theme({
  '&': { backgroundColor: 'var(--color-base-100)', color: 'var(--color-base-content)', height: '100%', fontSize: '13px' },
  '.cm-scroller': { fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace', lineHeight: '1.55' },
  '.cm-gutters': { backgroundColor: 'var(--color-base-200)', color: 'color-mix(in oklab, var(--color-base-content) 50%, transparent)', border: 'none' },
  '.cm-activeLine': { backgroundColor: 'color-mix(in oklab, var(--color-base-content) 5%, transparent)' },
  '.cm-activeLineGutter': { backgroundColor: 'color-mix(in oklab, var(--color-base-content) 8%, transparent)' },
  '.cm-cursor': { borderLeftColor: 'var(--color-base-content)' },
  '&.cm-focused': { outline: '2px solid var(--color-primary)', outlineOffset: '-2px' },
  '.cm-selectionBackground, &.cm-focused .cm-selectionBackground': { backgroundColor: 'color-mix(in oklab, var(--color-primary) 25%, transparent)' },
});

export const syntaxColors = syntaxHighlighting(
  HighlightStyle.define([
    { tag: [tags.keyword, tags.operatorKeyword, tags.modifier], color: 'var(--color-primary)', fontWeight: '600' },
    { tag: [tags.string, tags.special(tags.string)], color: 'var(--color-success)' },
    { tag: [tags.number, tags.bool, tags.null], color: 'var(--color-warning)' },
    { tag: [tags.comment, tags.meta], color: 'color-mix(in oklab, var(--color-base-content) 55%, transparent)', fontStyle: 'italic' },
    { tag: [tags.function(tags.variableName), tags.function(tags.propertyName), tags.tagName], color: 'var(--color-secondary)' },
    { tag: [tags.typeName, tags.className, tags.attributeName, tags.propertyName], color: 'var(--color-accent)' },
    { tag: tags.heading, fontWeight: '700' },
    { tag: tags.link, textDecoration: 'underline' },
  ]),
);
