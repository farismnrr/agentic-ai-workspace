<script lang="ts">
  import { NOT_CONNECTED } from '../content/user';

  interface Props {
    show: boolean;
  }

  const { show }: Props = $props();
</script>

{#if show}
  <div class="alert alert-info alert-soft" role="status">{NOT_CONNECTED}</div>
{/if}
