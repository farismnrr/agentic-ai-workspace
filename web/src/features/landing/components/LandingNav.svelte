<script lang="ts">
  import LogoMark from '../../../shared/ui/LogoMark.svelte';
  import { navLinks, REPO_URL } from '../content';
</script>

<header class="mx-auto w-full max-w-6xl px-6">
  <div class="navbar min-h-16 gap-4 px-0 sm:gap-6">
    <a class="flex min-h-11 shrink-0 items-center gap-2.5" href="/" aria-label="Masih Awam home">
      <LogoMark />
      <span class="whitespace-nowrap text-[17px] font-semibold tracking-[-0.01em] text-base-content">Masih Awam</span>
    </a>

    <nav class="hidden flex-1 items-center gap-7 md:flex" aria-label="Primary">
      {#each navLinks as link (link.href)}
        <a class="text-[15px] font-medium text-base-content/80 transition-colors hover:text-base-content" href={link.href}>{link.label}</a>
      {/each}
    </nav>

    <div class="ml-auto flex items-center gap-3 md:ml-0">
      <a class="text-[15px] font-medium text-base-content/80 transition-colors hover:text-base-content hidden sm:inline" href={REPO_URL}>GitHub</a>
      <a class="btn btn-primary h-10 min-h-10 rounded-full border-0 px-5 text-[15px] font-semibold text-primary-content hover:bg-[color-mix(in_oklab,var(--color-primary)_85%,black)]" href={REPO_URL}>Get started</a>
      <div class="dropdown dropdown-end md:hidden">
        <button type="button" class="btn btn-ghost btn-square min-h-11" aria-label="Open menu" aria-haspopup="true">
          <svg class="size-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
            <path d="M4 7h16M4 12h16M4 17h16" />
          </svg>
        </button>
        <ul class="menu dropdown-content z-10 mt-2 w-56 rounded-box border border-[var(--brand-border)] bg-base-100 p-2 shadow-[var(--brand-shadow)]">
          {#each navLinks as link (link.href)}
            <li><a class="min-h-11" href={link.href}>{link.label}</a></li>
          {/each}
          <li><a class="min-h-11" href={REPO_URL}>GitHub</a></li>
        </ul>
      </div>
    </div>
  </div>
</header>
