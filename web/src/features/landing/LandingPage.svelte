<script lang="ts">
  import AgentStage from './components/AgentStage.svelte';
  import CapabilityGrid from './components/CapabilityGrid.svelte';
  import LandingFooter from './components/LandingFooter.svelte';
  import LandingNav from './components/LandingNav.svelte';
  import WorkflowSection from './components/WorkflowSection.svelte';
</script>

<main class="overflow-hidden text-base-content">
  <div class="mx-auto w-full max-w-[1440px] px-5 sm:px-8 lg:px-12">
    <LandingNav />

    <section class="grid min-h-[calc(100svh-6rem)] items-center gap-14 py-14 lg:grid-cols-[0.92fr_1.08fr] lg:py-20">
      <div class="relative z-10 max-w-2xl">
        <div class="badge badge-soft badge-primary gap-2 rounded-full px-4 py-3 text-xs font-semibold uppercase tracking-[0.18em]">
          <span class="size-1.5 rounded-full bg-primary" aria-hidden="true"></span>
          Browser-native coding agent
        </div>

        <h1 class="mt-7 text-5xl font-semibold leading-[0.94] tracking-[-0.055em] sm:text-6xl lg:text-7xl xl:text-[5.5rem]">
          Give the web a repo.
          <span class="text-primary">Get working software back.</span>
        </h1>

        <p class="mt-7 max-w-xl text-base leading-7 text-[var(--brand-text-secondary)] sm:text-lg">
          Masih Awam is an agentic AI workspace for people who want Claude Code-style autonomy without living in a terminal.
          Set the goal, watch the agent inspect, edit, run, and verify—directly from the browser.
        </p>

        <div class="mt-9 flex flex-wrap items-center gap-3">
          <a class="btn btn-primary rounded-full px-6" href="#workspace-preview">See the agent work</a>
          <a class="btn btn-ghost rounded-full border border-[var(--brand-border)] px-6" href="#how-it-works">How it works</a>
        </div>

        <div class="mt-10 flex flex-wrap gap-x-7 gap-y-3 text-sm text-[var(--brand-text-tertiary)]">
          <span>Plans before edits</span>
          <span>Runs real tools</span>
          <span>Verifies the result</span>
        </div>
      </div>

      <AgentStage />
    </section>

    <div class="border-y border-[var(--brand-border)] py-5">
      <div class="grid gap-4 text-xs font-semibold uppercase tracking-[0.18em] text-[var(--brand-text-tertiary)] sm:grid-cols-4">
        <span>01 · understand</span>
        <span>02 · plan</span>
        <span>03 · execute</span>
        <span>04 · prove</span>
      </div>
    </div>

    <CapabilityGrid />
    <WorkflowSection />
  </div>

  <LandingFooter />
</main>
