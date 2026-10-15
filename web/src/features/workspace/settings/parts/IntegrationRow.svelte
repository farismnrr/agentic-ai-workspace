<script lang="ts">
  import type { Snippet } from 'svelte';
  import IntegrationLogo from './IntegrationLogo.svelte';
  import type { IntegrationName } from './integration-logos';

  interface Props {
    logo: IntegrationName;
    title: string;
    description: string;
    /** Shows the description as an error. */
    problem?: boolean | undefined;
    children: Snippet;
  }

  const { logo, title, description, problem = false, children }: Props = $props();
</script>

<li class="list-row items-center gap-4 p-4">
  <IntegrationLogo name={logo} />
  <div class="min-w-0">
    <p class="font-medium">{title}</p>
    <p class="mt-0.5 text-xs {problem ? 'text-error' : 'text-[var(--brand-text-secondary)]'}" role={problem ? 'alert' : undefined}>
      {description}
    </p>
  </div>
  <div class="flex flex-wrap items-center justify-end gap-2">{@render children()}</div>
</li>
