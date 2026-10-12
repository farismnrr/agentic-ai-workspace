<script lang="ts">
  import { APP_DESCRIPTION, APP_TITLE } from '../shared/config/app';
  import HelloCard from '../shared/ui/HelloCard.svelte';
  import ThemeToggle from '../shared/ui/ThemeToggle.svelte';
</script>

<main class="min-h-screen text-base-content">
  <div class="mx-auto flex min-h-screen w-full max-w-6xl flex-col px-5 py-5 sm:px-8 sm:py-8">
    <header
      class="flex items-center justify-between gap-4 rounded-full border border-[var(--brand-border)] bg-[var(--brand-surface-soft)] px-4 py-3 backdrop-blur-xl sm:px-5"
    >
      <div class="flex min-w-0 items-center gap-3">
        <span class="size-2.5 shrink-0 rounded-full bg-primary shadow-[0_0_24px_var(--color-primary)]" aria-hidden="true"></span>
        <div class="min-w-0 leading-tight">
          <p class="truncate text-sm font-semibold tracking-tight">Masih Awam</p>
          <p class="truncate text-xs text-[var(--brand-text-secondary)]">AI Workspace</p>
        </div>
      </div>
      <ThemeToggle />
    </header>

    <div class="flex flex-1 items-center justify-center py-10 sm:py-14">
      <HelloCard title={APP_TITLE} message={APP_DESCRIPTION} />
    </div>
  </div>
</main>
