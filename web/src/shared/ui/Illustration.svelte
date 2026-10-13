<script module lang="ts">
  export interface IllustrationAsset {
    src: string;
    width: number;
    height: number;
    alt: string;
    srcset?: string;
  }
</script>

<script lang="ts">
  interface Props {
    asset: IllustrationAsset;
    class?: string;
    priority?: boolean;
    sizes?: string;
  }

  const { asset, class: className = '', priority = false, sizes = '(min-width: 1024px) 320px, 80vw' }: Props = $props();
</script>

<img
  class={className}
  src={asset.src}
  srcset={asset.srcset}
  sizes={asset.srcset === undefined ? undefined : sizes}
  width={asset.width}
  height={asset.height}
  alt={asset.alt}
  loading={priority ? 'eager' : 'lazy'}
  fetchpriority={priority ? 'high' : 'auto'}
  decoding="async"
/>
