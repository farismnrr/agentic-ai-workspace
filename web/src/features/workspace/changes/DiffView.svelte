<script lang="ts">
  import { createDiff, type DiffEditor } from '../logic/diff-editor';

  interface Props {
    path: string;
    before: string;
    after: string;
  }

  const { path, before, after }: Props = $props();
  let host = $state<HTMLDivElement>();

  // The comparison is drawn again whenever the text changes.
  $effect(() => {
    const parent = host;
    if (parent === undefined) return;
    let gone = false;
    let made: DiffEditor | undefined;
    void createDiff({ parent, path, before, after }).then((created) => {
      if (gone) created.destroy();
      else made = created;
    });
    return () => {
      gone = true;
      made?.destroy();
    };
  });
</script>

<div bind:this={host} class="max-h-[32rem] overflow-auto" aria-label={`Changes in ${path}`}></div>
