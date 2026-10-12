<section id="proof" class="bg-base-100">
  <div class="mx-auto w-full max-w-[1600px] px-5 py-20 sm:px-8 lg:px-12 lg:py-28">
    <div class="grid gap-10 lg:grid-cols-[0.56fr_1.44fr] lg:items-start lg:gap-16">
      <div>
        <h2 class="max-w-lg text-4xl font-semibold tracking-[-0.04em] sm:text-5xl lg:text-6xl">Review the evidence.</h2>
        <p class="mt-5 max-w-md text-base leading-7 text-[var(--brand-text-secondary)] sm:text-lg">Completion is a diff and checks you can inspect, not a confident paragraph.</p>
      </div>

      <div class="min-h-[31rem] overflow-hidden border border-[var(--brand-border-strong)] bg-base-200">
        <div class="flex items-center justify-between border-b border-[var(--brand-border)] px-6 py-5"><span class="font-medium">callback.ts</span><span class="text-sm text-success">+7 −2</span></div>
        <div class="overflow-x-auto font-mono text-sm"><div class="grid min-w-[38rem] grid-cols-[3rem_1fr] gap-4 px-6 py-4 text-[var(--brand-text-secondary)]"><span>88</span><code>const redirect = parseRedirect(url);</code></div><div class="grid min-w-[38rem] grid-cols-[3rem_1fr] gap-4 bg-success/10 px-6 py-4"><span>89</span><code>+ const safe = allowRedirect(redirect);</code></div><div class="grid min-w-[38rem] grid-cols-[3rem_1fr] gap-4 bg-success/10 px-6 py-4"><span>90</span><code>+ return finishAuth(safe ?? '/');</code></div></div>
        <div class="mt-10 grid border-t border-[var(--brand-border)] sm:grid-cols-3">
          <div class="p-6 lg:p-7"><p class="text-sm text-[var(--brand-text-tertiary)]">Verification</p><p class="mt-2 text-2xl font-semibold">4 / 4</p><p class="mt-1 text-sm text-success">all checks passed</p></div>
          <div class="border-y border-[var(--brand-border)] p-6 sm:border-x sm:border-y-0 lg:p-7"><p class="text-sm text-[var(--brand-text-tertiary)]">Browser</p><p class="mt-2 text-2xl font-semibold">/workspace</p><p class="mt-1 text-sm text-success">redirect confirmed</p></div>
          <div class="p-6 lg:p-7"><p class="text-sm text-[var(--brand-text-tertiary)]">Release</p><p class="mt-2 text-2xl font-semibold">Latest</p><p class="mt-1 text-sm text-[var(--brand-text-secondary)]">binary ready</p></div>
        </div>
      </div>
    </div>
  </div>
</section>
