<script lang="ts">
  interface Props {
    title: string;
    message: string;
  }

  let { title, message }: Props = $props();
</script>

<section
  class="w-full max-w-3xl overflow-hidden rounded-[var(--radius-box)] border border-[var(--brand-border)] bg-base-100/88 shadow-[var(--brand-shadow)] backdrop-blur-xl"
>
  <div class="grid gap-0 lg:grid-cols-[1.35fr_0.65fr]">
    <div class="p-7 sm:p-10 lg:p-12">
      <div class="mb-8 flex flex-wrap items-center gap-2">
        <span class="badge badge-primary">Masih Awam</span>
        <span class="badge border-[var(--brand-border)] bg-[var(--brand-surface-soft)] text-base-content">Design system active</span>
      </div>

      <div class="space-y-4">
        <p class="text-xs font-bold uppercase tracking-[0.16em] text-[var(--brand-text-tertiary)]">Workspace foundation</p>
        <h1 class="max-w-2xl text-4xl font-semibold tracking-[-0.05em] text-balance sm:text-5xl lg:text-6xl">{title}</h1>
        <p class="max-w-xl text-base leading-7 text-[var(--brand-text-secondary)] sm:text-lg">{message}</p>
      </div>
    </div>

    <aside class="border-t border-[var(--brand-border)] bg-[var(--brand-surface-soft)] p-7 lg:border-t-0 lg:border-l sm:p-8">
      <p class="mb-5 text-xs font-bold uppercase tracking-[0.16em] text-[var(--brand-text-tertiary)]">Semantic roles</p>
      <div class="space-y-3 text-sm">
        <div class="flex items-center justify-between gap-4">
          <span class="text-[var(--brand-text-secondary)]">Primary</span>
          <span class="badge badge-primary">Brand</span>
        </div>
        <div class="flex items-center justify-between gap-4">
          <span class="text-[var(--brand-text-secondary)]">Secondary</span>
          <span class="badge badge-secondary">Agent</span>
        </div>
        <div class="flex items-center justify-between gap-4">
          <span class="text-[var(--brand-text-secondary)]">Tertiary</span>
          <span class="badge badge-accent">Context</span>
        </div>
        <div class="my-4 h-px bg-[var(--brand-border)]" aria-hidden="true"></div>
        <div class="flex flex-wrap gap-2">
          <span class="badge badge-success badge-soft">Success</span>
          <span class="badge badge-warning badge-soft">Warning</span>
          <span class="badge badge-error badge-soft">Error</span>
        </div>
      </div>
    </aside>
  </div>
</section>
