<script lang="ts">
  import { marks, type IntegrationName } from './integration-logos';

  interface Props {
    name: IntegrationName;
  }

  const { name }: Props = $props();
</script>

<span class="grid size-10 shrink-0 place-items-center rounded-box bg-base-200 text-base-content" aria-hidden="true">
  <svg viewBox="0 0 24 24" class="size-6" fill="currentColor"><path d={marks[name]} /></svg>
</span>
