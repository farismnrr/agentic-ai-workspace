<script lang="ts">
  interface Props {
    class?: string;
  }

  const { class: className = 'size-7' }: Props = $props();
</script>

<svg class={className} viewBox="0 0 28 28" fill="none" aria-hidden="true">
  <path d="M2 8 13 2v22L2 24z" class="fill-primary" />
  <path d="m15 2 11 5v7l-11 4z" class="fill-primary opacity-35" />
  <path d="M15 20l11-5v9H15z" class="fill-primary" />
</svg>
