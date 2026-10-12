<script lang="ts">
</script>

<section id="proof" class="bg-base-100">
  <div class="mx-auto w-full max-w-[1440px] px-5 py-24 sm:px-8 lg:px-12 lg:py-36">
    <div class="grid gap-12 lg:grid-cols-[0.72fr_1.28fr] lg:gap-20">
      <div><h2 class="max-w-lg text-4xl font-semibold tracking-[-0.04em] sm:text-6xl lg:text-7xl">Review the evidence.</h2><p class="mt-6 max-w-md text-base leading-7 text-[var(--brand-text-secondary)] sm:text-lg">Completion is a diff and a set of checks you can inspect, not a confident paragraph.</p></div>

      <div class="space-y-5">
        <div class="overflow-hidden rounded-3xl border border-[var(--brand-border-strong)] bg-base-200">
          <div class="flex items-center justify-between border-b border-[var(--brand-border)] px-5 py-4"><span class="font-medium">callback.ts</span><span class="text-sm text-success">+7 −2</span></div>
          <div class="font-mono text-xs sm:text-sm"><div class="grid grid-cols-[3rem_1fr] gap-4 px-5 py-3 text-[var(--brand-text-secondary)]"><span>88</span><code>const redirect = parseRedirect(url);</code></div><div class="grid grid-cols-[3rem_1fr] gap-4 bg-success/10 px-5 py-3"><span>89</span><code>+ const safe = allowRedirect(redirect);</code></div><div class="grid grid-cols-[3rem_1fr] gap-4 bg-success/10 px-5 py-3"><span>90</span><code>+ return finishAuth(safe ?? '/');</code></div></div>
        </div>

        <div class="grid gap-5 sm:grid-cols-2"><div class="rounded-3xl border border-[var(--brand-border)] bg-base-200 p-6"><p class="text-sm text-[var(--brand-text-tertiary)]">Verification</p><p class="mt-3 text-3xl font-semibold">4 / 4</p><p class="mt-2 text-sm text-success">all checks passed</p></div><div class="rounded-3xl border border-[var(--brand-border)] bg-base-200 p-6"><p class="text-sm text-[var(--brand-text-tertiary)]">Release</p><p class="mt-3 text-3xl font-semibold">Latest</p><p class="mt-2 text-sm text-[var(--brand-text-secondary)]">binary ready to run</p></div></div>
      </div>
    </div>
  </div>
</section>
