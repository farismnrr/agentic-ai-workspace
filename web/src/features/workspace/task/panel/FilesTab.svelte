<script lang="ts">
  import FileBrowser from '../../files/FileBrowser.svelte';
  import type { Change } from '../../logic/task-types';

  interface Props {
    projectId: string;
    changes: Change[];
  }

  const { projectId, changes }: Props = $props();
  const changed = $derived(new Set(changes.filter((change) => !change.undone).flatMap((change) => change.files)));
</script>

{#key projectId}<FileBrowser {projectId} {changed} height="h-[calc(100dvh-12rem)]" />{/key}
