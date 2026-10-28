<script lang="ts">
  import FileBrowser from '../../files/FileBrowser.svelte';

  interface Props {
    projectId: string;
  }

  const { projectId }: Props = $props();
</script>

{#key projectId}<FileBrowser {projectId} height="h-[calc(100dvh-12rem)]" />{/key}
