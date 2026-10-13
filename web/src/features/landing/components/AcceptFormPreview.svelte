<script lang="ts">
  interface Props {
    class?: string;
  }

  const { class: className = '' }: Props = $props();
</script>

<div
  class="card gap-2 border border-[var(--brand-border)] bg-base-100 p-3 text-xs shadow-[var(--brand-shadow)] {className}"
  aria-hidden="true"
  inert
>
  <span class="badge badge-sm w-full justify-start rounded-md border-primary bg-[var(--brand-marker)] text-base-content">Message sent ✓</span>
  <span class="rounded-md border border-[var(--brand-border-strong)] px-2 py-1 text-[var(--brand-text-secondary)]">Email</span>
  <span class="h-12 rounded-md border border-[var(--brand-border-strong)] px-2 py-1 text-[var(--brand-text-secondary)]">Message</span>
  <span class="badge badge-primary mx-auto rounded-full">Accept</span>
</div>
