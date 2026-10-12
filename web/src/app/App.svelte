<script lang="ts">
  import { APP_DESCRIPTION, APP_TITLE } from '../shared/config/app';
  import HelloCard from '../shared/ui/HelloCard.svelte';
  import ThemeToggle from '../shared/ui/ThemeToggle.svelte';
</script>

<main class="min-h-screen bg-base-200 text-base-content">
  <div class="mx-auto flex min-h-screen w-full max-w-6xl flex-col px-6 py-6 sm:px-8 sm:py-8">
    <header class="flex items-center justify-between gap-4">
      <div class="flex items-center gap-3 text-sm font-semibold tracking-tight">
        <span class="size-2 rounded-full bg-primary" aria-hidden="true"></span>
        <span>Masih Awam</span>
      </div>
      <ThemeToggle />
    </header>

    <div class="flex flex-1 items-center justify-center py-10">
      <HelloCard title={APP_TITLE} message={APP_DESCRIPTION} />
    </div>
  </div>
</main>
