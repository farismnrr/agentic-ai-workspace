<script lang="ts">
  import ThemeToggle from '../../../shared/ui/ThemeToggle.svelte';
</script>

<header class="flex h-24 items-center justify-between gap-5">
  <a class="group flex min-w-0 items-center gap-3" href="/" aria-label="Masih Awam home">
    <span
      class="grid size-10 shrink-0 place-items-center rounded-2xl border border-[var(--brand-border)] bg-[var(--brand-surface)] shadow-[var(--brand-shadow)]"
      aria-hidden="true"
    >
      <span class="size-3 -rotate-12 rounded-[0.3rem] bg-primary transition-transform duration-300 group-hover:rotate-12"></span>
    </span>
    <span class="min-w-0 leading-tight">
      <span class="block truncate text-sm font-semibold tracking-tight">Masih Awam</span>
      <span class="block truncate text-xs text-[var(--brand-text-tertiary)]">Agentic AI Workspace</span>
    </span>
  </a>

  <nav class="hidden items-center gap-7 text-sm text-[var(--brand-text-secondary)] md:flex" aria-label="Primary navigation">
    <a class="transition-colors hover:text-base-content" href="#capabilities">Capabilities</a>
    <a class="transition-colors hover:text-base-content" href="#how-it-works">How it works</a>
    <a class="transition-colors hover:text-base-content" href="#launch">Vision</a>
  </nav>

  <div class="flex items-center gap-2">
    <ThemeToggle />
    <a class="btn btn-neutral btn-sm hidden rounded-full sm:inline-flex" href="#launch">Enter workspace</a>
  </div>
</header>
