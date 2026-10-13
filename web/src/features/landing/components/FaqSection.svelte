<script lang="ts">
  import { reveal } from '../../../shared/motion/reveal';
  import Illustration from '../../../shared/ui/Illustration.svelte';
  import { illustrations } from '../assets/illustrations';
  import { faq } from '../content';
</script>

<section id="faq" class="mx-auto w-full max-w-6xl px-6 py-14 lg:py-20">
  <div class="grid gap-10 border-t border-[var(--brand-border-strong)] pt-14 lg:grid-cols-[1fr_1.4fr] lg:pt-16">
    <div use:reveal>
      <p class="text-sm text-[var(--brand-text-secondary)]">FAQ</p>
      <h2 class="mt-3 text-4xl font-medium leading-[1.05] tracking-tight sm:text-5xl">Questions,<br />answered.</h2>
      <Illustration asset={illustrations.paperAirplane} class="float-soft mt-8 hidden h-auto w-64 md:block" />
    </div>

    <div class="border-t border-[var(--brand-border-strong)]" use:reveal={120}>
      {#each faq as entry, index (entry.question)}
        <details class="collapse collapse-plus rounded-none border-b border-[var(--brand-border-strong)]" open={index === 0}>
          <summary class="collapse-title min-h-12 text-lg font-medium">{entry.question}</summary>
          <div class="collapse-content text-[var(--brand-text-secondary)]">
            <p class="max-w-md">{entry.answer}</p>
          </div>
        </details>
      {/each}
    </div>
  </div>
</section>
