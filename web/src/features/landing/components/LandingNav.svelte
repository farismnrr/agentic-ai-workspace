<script lang="ts">
  import LogoMark from '../../../shared/ui/LogoMark.svelte';
  import { navLinks, REPO_URL } from '../content';
</script>

<header class="mx-auto w-full max-w-6xl px-6">
  <div class="navbar min-h-20 gap-6 px-0">
    <a class="flex items-center gap-3 text-xl" href="/" aria-label="Masih Awam home">
      <LogoMark />
      <span class="font-medium tracking-tight">Masih Awam</span>
    </a>

    <nav class="hidden flex-1 items-center gap-7 md:flex" aria-label="Primary">
      {#each navLinks as link (link.href)}
        <a class="link link-hover" href={link.href}>{link.label}</a>
      {/each}
    </nav>

    <div class="ml-auto flex items-center gap-3 md:ml-0">
      <a class="link link-hover hidden sm:inline" href={REPO_URL}>GitHub</a>
      <a class="btn btn-primary rounded-full" href={REPO_URL}>Get started</a>
    </div>
  </div>
</header>
