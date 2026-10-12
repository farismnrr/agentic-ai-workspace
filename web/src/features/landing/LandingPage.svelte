<script lang="ts">
  import CloserLookSection from './components/CloserLookSection.svelte';
  import DelegateSection from './components/DelegateSection.svelte';
  import HighlightsSection from './components/HighlightsSection.svelte';
  import LandingFooter from './components/LandingFooter.svelte';
  import ParallelSection from './components/ParallelSection.svelte';
  import RuntimeSection from './components/RuntimeSection.svelte';
  import StoryHero from './components/StoryHero.svelte';
</script>

<main class="overflow-x-clip bg-base-200 text-base-content">
  <StoryHero />
  <HighlightsSection />
  <CloserLookSection />
  <DelegateSection />
  <ParallelSection />
  <RuntimeSection />

  <section id="launch" class="bg-base-100">
    <div class="mx-auto w-full max-w-[1600px] px-5 py-20 sm:px-8 lg:px-12 lg:py-24">
      <div class="flex flex-col gap-8 border-y border-[var(--brand-border-strong)] py-12 sm:flex-row sm:items-end sm:justify-between lg:py-14">
        <div>
          <h2 class="max-w-3xl text-4xl font-semibold tracking-[-0.04em] sm:text-6xl lg:text-7xl">Open the orchestrator.</h2>
          <p class="mt-4 max-w-xl text-base leading-7 text-[var(--brand-text-secondary)] sm:text-lg">Give it a software task and keep every agent, change, command, and check in view.</p>
        </div>
        <a class="btn btn-primary btn-lg shrink-0" href="/">Open workspace</a>
      </div>
    </div>
  </section>

  <LandingFooter />
</main>
