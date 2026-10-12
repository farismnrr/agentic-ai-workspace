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

  <section id="launch" class="mx-auto w-full max-w-[1440px] px-5 pb-28 sm:px-8 lg:px-12 lg:pb-36">
    <div class="flex flex-col gap-8 border-t border-[var(--brand-border-strong)] py-14 sm:flex-row sm:items-center sm:justify-between lg:py-16">
      <h2 class="text-3xl font-semibold tracking-[-0.03em] sm:text-4xl lg:text-5xl">Open the workspace.</h2>
      <a class="btn btn-primary sm:px-7" href="/">Open workspace</a>
    </div>
  </section>

  <LandingFooter />
</main>
