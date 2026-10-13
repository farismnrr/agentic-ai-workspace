<script lang="ts">
  interface Props {
    class?: string;
  }

  const { class: className = 'size-8' }: Props = $props();
</script>

<svg class={className} viewBox="0 0 64 64" aria-hidden="true">
  <rect width="64" height="64" rx="16.64" class="fill-primary" />
  <g transform="translate(9.6,21.56) scale(0.357) translate(2,-29.5)">
    <g fill="none" stroke-width="11" stroke-linecap="round" class="stroke-primary-content">
      <ellipse cx="18" cy="56" rx="14.5" ry="21" />
      <path d="M88,40 V77.5 M88,58 C88,46 94,40 102,40 C110,40 116,46 116,58 V77.5" />
    </g>
    <g class="fill-primary-content">
      <circle cx="49" cy="76.5" r="6.5" />
      <circle cx="67" cy="76.5" r="6.5" />
    </g>
  </g>
</svg>
