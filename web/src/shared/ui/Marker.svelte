<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    children: Snippet;
  }

  const { children }: Props = $props();
</script>

<span
  class="box-decoration-clone px-1 [background:linear-gradient(transparent_52%,var(--brand-marker)_52%,var(--brand-marker)_90%,transparent_90%)]"
>
  {@render children()}
</span>
