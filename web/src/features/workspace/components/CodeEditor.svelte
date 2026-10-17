<script lang="ts">
  import { onMount } from 'svelte';
  import { createEditor, type Editor } from '../logic/editor';

  interface Props {
    path: string;
    /** The text to show; when it changes from outside (not by typing), the editor takes it over. */
    value: string;
    readOnly?: boolean;
    onchange: (text: string) => void;
  }

  const { path, value, readOnly = false, onchange }: Props = $props();
  let host = $state<HTMLDivElement>();
  let editor = $state<Editor>();

  onMount(() => {
    let gone = false;
    let made: Editor | undefined;
    if (host !== undefined) {
      void createEditor({ parent: host, path, doc: value, readOnly, onchange }).then((created) => {
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
</script>

<div bind:this={host} class="h-full min-h-0 overflow-hidden" aria-label={`Editor for ${path}`}></div>
