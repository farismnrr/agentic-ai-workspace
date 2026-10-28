<script lang="ts">
  import { onMount } from 'svelte';
  import { createEditor, type Editor } from '../logic/editor';

  interface Props {
    path: string;
    /** The text to show; when it changes from outside (not by typing), the editor takes it over. */
    value: string;
    /** Current Git HEAD text, omitted when Git cannot provide a safe baseline. */
    original?: string | undefined;
    readOnly?: boolean;
    onchange: (text: string) => void;
  }

  const { path, value, original, readOnly = false, onchange }: Props = $props();
  let host = $state<HTMLDivElement>();
  let editor = $state<Editor>();

  onMount(() => {
    let gone = false;
    let made: Editor | undefined;
    if (host !== undefined) {
      void createEditor({ parent: host, path, doc: value, original, readOnly, onchange }).then((created) => {
        if (gone) created.destroy();
        else editor = made = created;
      });
    }
    return () => {
      gone = true;
      made?.destroy();
    };
  });

  $effect(() => {
    if (editor !== undefined && editor.text() !== value) editor.replace(value);
  });

  $effect(() => {
    if (editor !== undefined) editor.setGitBaseline(original);
  });
</script>

<div bind:this={host} class="h-full min-h-0 overflow-hidden" aria-label={`Editor for ${path}`}></div>
