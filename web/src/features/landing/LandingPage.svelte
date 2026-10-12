<script lang="ts">
  import AgentGraph from './components/AgentGraph.svelte';
  import LandingFooter from './components/LandingFooter.svelte';
  import LandingNav from './components/LandingNav.svelte';
  import NarrativeSection from './components/NarrativeSection.svelte';
  import RuntimeSection from './components/RuntimeSection.svelte';
</script>

<main class="overflow-hidden text-base-content">
  <section class="relative min-h-[100svh] overflow-hidden border-b border-[var(--brand-border)]">
    <AgentGraph />

    <div class="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_50%_45%,transparent_0%,var(--color-base-200)_74%)] opacity-80" aria-hidden="true"></div>
    <div class="pointer-events-none absolute inset-x-0 bottom-0 h-1/2 bg-gradient-to-t from-base-200 via-base-200/55 to-transparent" aria-hidden="true"></div>

    <div class="relative z-20 mx-auto flex min-h-[100svh] w-full max-w-[1440px] flex-col px-5 sm:px-8 lg:px-12">
      <LandingNav />

      <div class="mt-auto grid items-end gap-10 pb-8 pt-28 sm:pb-12 lg:grid-cols-[1fr_auto] lg:pb-14">
        <div class="max-w-4xl">
          <p class="font-mono text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-primary">Masih Awam / Agentic runtime 001</p>
          <h1 class="mt-5 text-5xl font-semibold leading-[0.88] tracking-[-0.06em] sm:text-7xl lg:text-[6.6rem] xl:text-[7.4rem]">
            Code doesn’t need
            <span class="block text-[var(--brand-text-tertiary)]">another chat window.</span>
          </h1>
          <div class="mt-7 flex flex-col gap-7 sm:flex-row sm:items-end sm:justify-between">
            <p class="max-w-xl text-base leading-7 text-[var(--brand-text-secondary)] sm:text-lg">
              One browser tab. One goal. A coding agent that reads the repo, edits files, runs commands, and proves the result while you watch the system move.
            </p>
            <div class="flex shrink-0 gap-3">
              <a class="btn btn-primary rounded-none px-6" href="#how-it-works">Watch it operate</a>
              <a class="btn btn-ghost rounded-none border border-[var(--brand-border-strong)] px-6" href="#capabilities">Inspect system</a>
            </div>
          </div>
        </div>

        <div class="hidden min-w-44 border-l border-[var(--brand-border-strong)] pl-5 font-mono text-[0.65rem] uppercase tracking-[0.18em] lg:block">
          <p class="mb-4 text-[var(--brand-text-tertiary)]">Live loop</p>
          <div class="space-y-2.5">
            <p><span class="mr-3 text-secondary">01</span>read</p>
            <p><span class="mr-3 text-primary">02</span>reason</p>
            <p><span class="mr-3 text-accent">03</span>execute</p>
            <p><span class="mr-3 text-success">04</span>verify</p>
          </div>
        </div>
      </div>
    </div>
  </section>

  <NarrativeSection />
  <RuntimeSection />

  <section id="launch" class="mx-auto w-full max-w-[1440px] px-5 pb-28 sm:px-8 lg:px-12 lg:pb-40">
    <div class="grid gap-8 border-y border-[var(--brand-border-strong)] py-12 lg:grid-cols-[1fr_auto] lg:items-end lg:py-16">
      <div>
        <p class="font-mono text-xs font-semibold uppercase tracking-[0.24em] text-primary">System / 03</p>
        <h2 class="mt-5 max-w-4xl text-5xl font-semibold leading-[0.92] tracking-[-0.055em] sm:text-7xl">
          Your repo is the interface.
        </h2>
      </div>
      <a class="btn btn-primary rounded-none px-8" href="/">Enter workspace</a>
    </div>
  </section>

  <LandingFooter />
</main>
