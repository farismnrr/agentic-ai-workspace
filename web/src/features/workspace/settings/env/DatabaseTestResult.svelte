<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';
  import type { TestResult } from '../env-api';

  interface Props { result: TestResult | undefined }
  const { result }: Props = $props();
</script>

{#if result}
  <p class="flex items-center gap-2 text-sm" class:text-error={!result.ok}>
    <Icon name={result.ok ? 'check-circle' : 'alert-triangle'} class="size-4 shrink-0" />{result.message}
  </p>
{/if}
