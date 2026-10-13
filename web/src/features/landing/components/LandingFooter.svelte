<script lang="ts">
  import LogoMark from '../../../shared/ui/LogoMark.svelte';
  import { footerLinks } from '../content';
</script>

<footer class="mx-auto w-full max-w-6xl px-6 pb-10">
  <div class="flex flex-col gap-6 border-t border-[var(--brand-border-strong)] pt-8 sm:flex-row sm:items-start sm:justify-between">
    <a class="flex items-center gap-2.5" href="/" aria-label="Masih Awam home">
      <LogoMark />
      <span class="leading-tight">
        <span class="block text-lg font-semibold tracking-[-0.01em]">Masih Awam</span>
        <span class="block text-sm text-[var(--brand-text-secondary)]">Work you can trust.</span>
      </span>
    </a>
    <div class="flex flex-col gap-3 sm:items-end">
      <nav class="flex flex-wrap gap-x-6 gap-y-2" aria-label="Footer">
        {#each footerLinks as link (link.href)}
          <a class="link link-hover" href={link.href}>{link.label}</a>
        {/each}
      </nav>
      <p class="text-sm text-[var(--brand-text-secondary)]">© 2026 Masih Awam</p>
    </div>
  </div>
</footer>
