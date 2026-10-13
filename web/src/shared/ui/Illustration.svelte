<script module lang="ts">
  export interface IllustrationAsset {
    src: string;
    width: number;
    height: number;
    alt: string;
  }
</script>

<script lang="ts">
  interface Props {
    asset: IllustrationAsset;
    class?: string;
    priority?: boolean;
  }

  const { asset, class: className = '', priority = false }: Props = $props();
</script>

<img
  class={className}
  src={asset.src}
  width={asset.width}
  height={asset.height}
  alt={asset.alt}
  loading={priority ? 'eager' : 'lazy'}
  fetchpriority={priority ? 'high' : 'auto'}
  decoding="async"
/>
