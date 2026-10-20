<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';
  import { extensionsCopy } from '../../content/extensions-copy';

  interface Props {
    text: string;
    url: string;
  }

  const { text, url }: Props = $props();
</script>

<section class="card flex-row flex-wrap items-center gap-3 border border-primary/40 bg-base-100 p-3" aria-label="Browser extension">
  <Icon name="puzzle" class="size-5 shrink-0 text-primary" />
  <div class="min-w-0 flex-1">
    <p class="font-medium">{text}</p>
    <p class="text-xs text-[var(--brand-text-secondary)]">{extensionsCopy.intro}</p>
  </div>
  <a class="btn btn-primary btn-sm rounded-lg" href={url}>{extensionsCopy.card.install}</a>
</section>
