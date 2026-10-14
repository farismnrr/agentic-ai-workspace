<script lang="ts">
  import TaskScreen from './TaskScreen.svelte';

  interface Props {
    projectId: string;
    taskId: string;
  }

  const { projectId, taskId }: Props = $props();
</script>

<!-- Keyed so that opening another task starts a fresh thread instead of reusing the old one. -->
{#key taskId}<TaskScreen {projectId} {taskId} />{/key}
