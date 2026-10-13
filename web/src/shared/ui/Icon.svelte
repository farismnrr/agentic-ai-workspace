<script module lang="ts">
  export type IconName = 'check' | 'chat' | 'heart' | 'laptop' | 'arrow-right' | 'eye' | 'eye-off';
</script>

<script lang="ts">
  interface Props {
    name: IconName;
    class?: string;
  }

  const paths: Record<IconName, string> = {
    check: 'M20 6 9 17l-5-5',
    chat: 'M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z',
    heart: 'M19 14c1.5-1.5 3-3.2 3-5.5A5.5 5.5 0 0 0 16.5 3c-1.8 0-3 .5-4.5 2-1.5-1.5-2.7-2-4.5-2A5.5 5.5 0 0 0 2 8.5c0 2.3 1.5 4 3 5.5l7 7z',
    laptop: 'M3 5h18v11H3zM2 20h20',
    'arrow-right': 'M5 12h14m-6-6 6 6-6 6',
    eye: 'M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12zM12 9a3 3 0 1 0 0 6 3 3 0 0 0 0-6z',
    'eye-off': 'M3 3l18 18M10.6 5.1A9.8 9.8 0 0 1 12 5c6.5 0 10 7 10 7a17 17 0 0 1-3.2 4.2M6.6 6.6C3.7 8.5 2 12 2 12s3.5 7 10 7a9.7 9.7 0 0 0 4.3-1M9.9 9.9a3 3 0 0 0 4.2 4.2',
  };

  const { name, class: className = 'size-5' }: Props = $props();
</script>

<svg
  class={className}
  viewBox="0 0 24 24"
  fill="none"
  stroke="currentColor"
  stroke-width="2"
  stroke-linecap="round"
  stroke-linejoin="round"
  aria-hidden="true"
>
  <path d={paths[name]} />
</svg>
