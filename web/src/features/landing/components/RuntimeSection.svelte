<script lang="ts">
  const trace = [
    { label: 'read repository rules', meta: 'complete' },
    { label: 'map relevant code', meta: 'complete' },
    { label: 'apply focused change', meta: 'complete' },
    { label: 'run verification', meta: 'passing' },
  ] as const;
</script>

<section id="how-it-works" class="mx-auto w-full max-w-[1440px] px-5 pb-28 sm:px-8 lg:px-12 lg:pb-40">
  <div class="overflow-hidden rounded-[2.5rem] border border-[var(--brand-border)] bg-base-100 shadow-[var(--brand-shadow)]">
    <div class="grid min-h-[44rem] lg:grid-cols-[0.88fr_1.12fr]">
      <div class="flex flex-col justify-between border-b border-[var(--brand-border)] p-7 sm:p-10 lg:border-b-0 lg:border-r lg:p-14">
        <div>
          <p class="font-mono text-xs font-semibold uppercase tracking-[0.24em] text-primary">Execution / visible</p>
          <h2 class="mt-6 max-w-lg text-5xl font-semibold leading-[0.92] tracking-[-0.055em] sm:text-6xl">
            Watch the work,
            <span class="text-[var(--brand-text-tertiary)]">not the magic trick.</span>
          </h2>
          <p class="mt-7 max-w-md leading-7 text-[var(--brand-text-secondary)]">
            Every useful step stays legible: what the agent read, what changed, what ran, and what finally proved the result.
          </p>
        </div>

        <div class="mt-16 border-t border-[var(--brand-border)]">
          {#each trace as item, index (item.label)}
            <div class="grid grid-cols-[3rem_1fr_auto] items-center gap-4 border-b border-[var(--brand-border)] py-4">
              <span class="font-mono text-xs text-[var(--brand-text-tertiary)]">0{index + 1}</span>
              <span class="text-sm font-medium">{item.label}</span>
              <span class="text-xs text-success">{item.meta}</span>
            </div>
          {/each}
        </div>
      </div>

      <div class="relative flex flex-col p-7 sm:p-10 lg:p-14">
        <div class="flex items-center justify-between border-b border-[var(--brand-border)] pb-4 font-mono text-[0.68rem] uppercase tracking-[0.18em] text-[var(--brand-text-tertiary)]">
          <span>workspace / landing</span>
          <span class="text-success">verified</span>
        </div>

        <div class="mt-10 rounded-[1.75rem] border border-[var(--brand-border)] bg-base-200 p-5 sm:p-7">
          <p class="font-mono text-xs uppercase tracking-[0.18em] text-[var(--brand-text-tertiary)]">Goal</p>
          <p class="mt-3 max-w-xl text-2xl font-semibold tracking-[-0.035em] sm:text-3xl">
            “Make the landing page feel intentional, cinematic, and unmistakably agentic.”
          </p>
        </div>

        <div class="mt-5 grid gap-5 sm:grid-cols-2">
          <div class="rounded-[1.75rem] border border-[var(--brand-border)] bg-base-200 p-5 sm:p-6">
            <p class="font-mono text-xs uppercase tracking-[0.18em] text-secondary">Change</p>
            <div class="mt-5 space-y-3 text-sm text-[var(--brand-text-secondary)]">
              <p>− decorative network graph</p>
              <p class="text-base-content">+ scroll-driven product story</p>
              <p class="text-base-content">+ semantic light / dark rendering</p>
            </div>
          </div>

          <div class="rounded-[1.75rem] border border-[var(--brand-border)] bg-base-200 p-5 sm:p-6">
            <p class="font-mono text-xs uppercase tracking-[0.18em] text-primary">Proof</p>
            <div class="mt-5 grid gap-3 text-sm text-[var(--brand-text-secondary)]">
              <span>✓ svelte-check</span>
              <span>✓ eslint</span>
              <span>✓ tests</span>
              <span>✓ production build</span>
            </div>
          </div>
        </div>

        <div class="mt-auto pt-12">
          <p class="max-w-2xl text-3xl font-semibold leading-tight tracking-[-0.04em] sm:text-4xl">
            The result is not a response. It is a change you can inspect.
          </p>
        </div>
      </div>
    </div>
  </div>
</section>
