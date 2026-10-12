<script lang="ts">
  const steps = [
    { index: '01', label: 'inspect', detail: 'map repo + rules' },
    { index: '02', label: 'change', detail: 'edit the smallest surface' },
    { index: '03', label: 'run', detail: 'execute real commands' },
    { index: '04', label: 'prove', detail: 'tests + build + checks' },
  ] as const;
</script>

<section id="how-it-works" class="mx-auto w-full max-w-[1440px] px-5 pb-28 sm:px-8 lg:px-12 lg:pb-40">
  <div class="overflow-hidden rounded-[2.25rem] bg-neutral text-neutral-content shadow-[var(--brand-shadow)]">
    <div class="grid min-h-[42rem] lg:grid-cols-[0.82fr_1.18fr]">
      <div class="flex flex-col justify-between border-b border-neutral-content/10 p-7 sm:p-10 lg:border-b-0 lg:border-r lg:p-14">
        <div>
          <p class="font-mono text-xs font-semibold uppercase tracking-[0.24em] text-primary">System / 02</p>
          <h2 class="mt-6 max-w-lg text-5xl font-semibold leading-[0.92] tracking-[-0.055em] sm:text-6xl">
            Execution,
            <span class="text-neutral-content/45">visible.</span>
          </h2>
        </div>

        <div class="mt-16 border-t border-neutral-content/12">
          {#each steps as step (step.index)}
            <div class="grid grid-cols-[3rem_1fr] gap-4 border-b border-neutral-content/10 py-4">
              <span class="font-mono text-xs text-neutral-content/35">{step.index}</span>
              <div class="flex items-baseline justify-between gap-4">
                <span class="font-medium">{step.label}</span>
                <span class="text-xs text-neutral-content/40">{step.detail}</span>
              </div>
            </div>
          {/each}
        </div>
      </div>

      <div class="relative overflow-hidden p-7 sm:p-10 lg:p-14">
        <div class="runtime-grid absolute inset-0 opacity-30" aria-hidden="true"></div>
        <div class="scan-line absolute inset-x-0 top-0 h-px bg-primary/80 shadow-[0_0_32px_var(--color-primary)]" aria-hidden="true"></div>

        <div class="relative flex h-full flex-col">
          <div class="flex items-center justify-between border-b border-neutral-content/10 pb-4 font-mono text-[0.68rem] uppercase tracking-[0.18em] text-neutral-content/40">
            <span>agent / live trace</span>
            <span class="text-success">● executing</span>
          </div>

          <div class="mt-9 space-y-8 font-mono text-sm leading-7">
            <div>
              <p class="text-neutral-content/35">$ inspect repository</p>
              <p class="mt-2 text-secondary">↳ architecture mapped · 42 files</p>
            </div>
            <div>
              <p class="text-neutral-content/35">$ apply change --scope landing</p>
              <div class="mt-3 border-l border-accent/50 pl-4 text-neutral-content/65">
                <p><span class="text-error">−</span> generic dashboard mockup</p>
                <p><span class="text-success">+</span> interactive repository graph</p>
                <p><span class="text-success">+</span> semantic motion system</p>
              </div>
            </div>
            <div>
              <p class="text-neutral-content/35">$ make verify</p>
              <div class="mt-3 grid gap-2 text-neutral-content/65 sm:grid-cols-2">
                <span>✓ svelte-check</span>
                <span>✓ eslint</span>
                <span>✓ tests</span>
                <span>✓ production build</span>
              </div>
            </div>
          </div>

          <div class="mt-auto pt-14">
            <p class="font-mono text-[0.65rem] uppercase tracking-[0.22em] text-neutral-content/35">Result</p>
            <div class="mt-3 flex items-end justify-between gap-6">
              <p class="max-w-lg text-2xl font-medium tracking-[-0.03em]">The agent does not stop at an answer. It stops at evidence.</p>
              <span class="hidden text-5xl font-semibold tracking-[-0.06em] text-primary sm:block">04/04</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</section>

<style>
  .runtime-grid {
    background-image:
      linear-gradient(rgb(255 255 255 / 7%) 1px, transparent 1px),
      linear-gradient(90deg, rgb(255 255 255 / 7%) 1px, transparent 1px);
    background-size: 42px 42px;
    mask-image: linear-gradient(to bottom, black, transparent 88%);
  }

  .scan-line {
    animation: scan 7s linear infinite;
  }

  @keyframes scan {
    0% { transform: translateY(3rem); opacity: 0; }
    12% { opacity: 0.8; }
    88% { opacity: 0.35; }
    100% { transform: translateY(38rem); opacity: 0; }
  }
</style>
