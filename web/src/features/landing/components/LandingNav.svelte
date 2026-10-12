<script lang="ts">
  import ThemeToggle from '../../../shared/ui/ThemeToggle.svelte';
</script>

<header class="flex h-24 items-center justify-between gap-6 border-b border-[var(--brand-border)]">
  <a class="group flex min-w-0 items-center gap-3" href="/" aria-label="Masih Awam home">
    <span class="grid size-5 shrink-0 grid-cols-2 gap-0.5 transition-transform duration-300 group-hover:rotate-45" aria-hidden="true">
      <span class="bg-primary"></span>
      <span class="bg-secondary"></span>
      <span class="bg-accent"></span>
      <span class="bg-base-content"></span>
    </span>
    <span class="min-w-0 leading-tight">
      <span class="block truncate text-sm font-semibold tracking-tight">Masih Awam</span>
      <span class="block truncate font-mono text-[0.62rem] uppercase tracking-[0.16em] text-[var(--brand-text-tertiary)]">Agentic workspace</span>
    </span>
  </a>

  <nav class="hidden items-center gap-8 font-mono text-[0.68rem] uppercase tracking-[0.16em] text-[var(--brand-text-secondary)] md:flex" aria-label="Primary navigation">
    <a class="transition-colors hover:text-base-content" href="#capabilities">System</a>
    <a class="transition-colors hover:text-base-content" href="#how-it-works">Execution</a>
    <a class="transition-colors hover:text-base-content" href="#launch">Workspace</a>
  </nav>

  <div class="flex items-center gap-2">
    <ThemeToggle />
    <a class="btn btn-neutral btn-sm hidden rounded-none sm:inline-flex" href="#launch">Open workspace</a>
  </div>
</header>
