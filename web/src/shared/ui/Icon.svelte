<script module lang="ts">
  export type IconName = 'check' | 'chat' | 'heart' | 'laptop' | 'arrow-right';
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
