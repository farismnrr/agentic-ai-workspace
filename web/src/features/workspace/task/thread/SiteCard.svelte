<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';
  import { sitesCopy } from '../../content/sites-copy';

  interface Props {
    text: string;
    url: string;
  }

  const { text, url }: Props = $props();
</script>

<section class="card flex-row flex-wrap items-center gap-3 border border-success/40 bg-base-100 p-3" aria-label="Running app">
  <Icon name="globe" class="size-5 shrink-0 text-success" />
  <div class="min-w-0 flex-1">
    <p class="font-medium">{text}</p>
    <p class="text-xs text-[var(--brand-text-secondary)]">{sitesCopy.card.privacy} <a class="link" href="/app/sites">{sitesCopy.card.manage}</a></p>
  </div>
  <a class="btn btn-primary btn-sm rounded-lg" href={url} target="_blank" rel="noopener noreferrer">{sitesCopy.open} <Icon name="external-link" class="size-4" /></a>
</section>
