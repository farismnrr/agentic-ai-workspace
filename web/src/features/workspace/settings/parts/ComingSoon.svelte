<script lang="ts">
  import { comingSoon } from '../../content/models';

  interface Props {
    title: string;
    description: string;
  }

  const { title, description }: Props = $props();
</script>

<section class="rounded-xl border border-dashed border-[var(--brand-border)] bg-base-100 px-4 py-6 text-center">
  <span class="badge badge-ghost badge-sm">{comingSoon.label}</span>
  <h2 class="mt-2 text-sm font-semibold">{title}</h2>
  <p class="mx-auto mt-1 max-w-sm text-xs text-[var(--brand-text-secondary)]">{description}</p>
</section>
