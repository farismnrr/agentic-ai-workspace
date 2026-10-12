<script lang="ts">
  import LandingFooter from './components/LandingFooter.svelte';
  import NarrativeSection from './components/NarrativeSection.svelte';
  import RuntimeSection from './components/RuntimeSection.svelte';
  import StoryHero from './components/StoryHero.svelte';
</script>

<main class="overflow-x-clip bg-base-200 text-base-content">
  <StoryHero />
  <NarrativeSection />
  <RuntimeSection />

  <section id="launch" class="mx-auto w-full max-w-[1440px] px-5 pb-28 sm:px-8 lg:px-12 lg:pb-40">
    <div class="grid gap-10 border-y border-[var(--brand-border-strong)] py-14 lg:grid-cols-[1fr_auto] lg:items-end lg:py-20">
      <div>
        <p class="font-mono text-xs font-semibold uppercase tracking-[0.24em] text-primary">Ready when you are</p>
        <h2 class="mt-5 max-w-4xl text-5xl font-semibold leading-[0.92] tracking-[-0.055em] sm:text-7xl">
          One goal. One workspace. Ship.
        </h2>
      </div>
      <a class="btn btn-primary px-8" href="/">Open workspace</a>
    </div>
  </section>

  <LandingFooter />
</main>
