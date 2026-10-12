<script lang="ts">
</script>

<section id="highlights" class="bg-base-100">
  <div class="mx-auto w-full max-w-[1600px] px-5 py-20 sm:px-8 lg:px-12 lg:py-28">
    <div class="max-w-4xl">
      <h2 class="text-4xl font-semibold tracking-[-0.04em] sm:text-5xl lg:text-6xl">See the work, not just the answer.</h2>
      <p class="mt-4 max-w-2xl text-base leading-7 text-[var(--brand-text-secondary)] sm:text-lg">Repository context, execution, and verification stay visible from the first read to the final check.</p>
    </div>

    <article class="mt-14 border-t border-[var(--brand-border-strong)] pt-10">
      <div class="mb-8 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <h3 class="max-w-2xl text-3xl font-semibold tracking-[-0.035em] sm:text-4xl">It reads the repo before it routes work.</h3>
        <p class="max-w-md leading-7 text-[var(--brand-text-secondary)]">Rules and relevant files become working context first.</p>
      </div>
      <div class="overflow-hidden rounded-[1.35rem] border border-[var(--brand-border)] bg-base-200">
        <div class="grid min-h-[28rem] lg:grid-cols-[0.72fr_1.28fr]">
          <div class="border-b border-[var(--brand-border)] p-7 font-mono text-sm leading-8 lg:border-b-0 lg:border-r lg:p-9">
            <p class="font-sans text-sm font-medium text-base-content">Repository</p>
            <p class="mt-6 text-primary">AGENTS.md</p><p>web/</p><p class="pl-4">src/</p><p class="pl-8">features/auth/</p><p class="pl-8 text-base-content">callback.ts</p><p class="pl-8">session.ts</p><p>tests/</p>
          </div>
          <div class="grid content-center p-8 sm:p-12 lg:p-16">
            <p class="max-w-xl text-3xl font-medium tracking-[-0.025em] sm:text-4xl">Read repository rules before changing code.</p>
            <div class="mt-10 grid gap-4 text-sm text-[var(--brand-text-secondary)] sm:grid-cols-3"><p>✓ reuse existing UI</p><p>✓ preserve both themes</p><p>✓ verify before done</p></div>
          </div>
        </div>
      </div>
    </article>

    <article class="mt-16 overflow-hidden rounded-[1.5rem] bg-neutral text-neutral-content">
      <div class="grid gap-6 border-b border-neutral-content/10 p-7 sm:p-10 lg:grid-cols-[0.8fr_1.2fr] lg:items-end lg:p-12">
        <h3 class="max-w-2xl text-3xl font-semibold tracking-[-0.035em] sm:text-4xl">Edits and commands stay side by side.</h3>
        <p class="max-w-xl leading-7 text-neutral-content/65 lg:justify-self-end">The task never disappears behind an “AI is working” screen.</p>
      </div>
      <div class="grid min-h-[30rem] lg:grid-cols-[1.12fr_0.88fr]">
        <div class="border-b border-neutral-content/10 lg:border-b-0 lg:border-r">
          <div class="border-b border-neutral-content/10 px-6 py-4 font-mono text-xs text-neutral-content/55">callback.ts · modified</div>
          <div class="overflow-x-auto p-6 font-mono text-sm leading-8 sm:p-8"><p class="whitespace-nowrap">41&nbsp; const next = params.get('next');</p><p class="whitespace-nowrap bg-success/10 text-success">42&nbsp; + const target = safeRedirect(next);</p><p class="whitespace-nowrap bg-success/10 text-success">43&nbsp; + await session.persist();</p><p class="whitespace-nowrap bg-success/10 text-success">44&nbsp; + return redirect(target);</p></div>
        </div>
        <div>
          <div class="border-b border-neutral-content/10 px-6 py-4 font-mono text-xs text-neutral-content/55">Terminal</div>
          <div class="overflow-x-auto p-6 font-mono text-sm leading-8 sm:p-8"><p class="whitespace-nowrap"><span class="text-primary">$</span> make verify</p><p class="whitespace-nowrap text-success">✓ frontend · passed</p><p class="whitespace-nowrap text-success">✓ tests · passed</p><p class="whitespace-nowrap text-success">✓ production build · passed</p></div>
        </div>
      </div>
    </article>

    <article class="mt-16 border-t border-[var(--brand-border-strong)] pt-10">
      <div class="mb-8 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <h3 class="max-w-2xl text-3xl font-semibold tracking-[-0.035em] sm:text-4xl">Done means the browser and checks agree.</h3>
        <p class="max-w-md leading-7 text-[var(--brand-text-secondary)]">A diff, a running product, and verification close the task.</p>
      </div>
      <div class="overflow-hidden rounded-[1.35rem] border border-[var(--brand-border)] bg-base-200">
        <div class="flex items-center gap-2 border-b border-[var(--brand-border)] px-5 py-4 text-xs text-[var(--brand-text-tertiary)]"><span class="ml-1">127.0.0.1:8080/auth/callback</span></div>
        <div class="grid min-h-[28rem] place-items-center p-8 text-center"><div><span class="text-4xl text-success">✓</span><p class="mt-5 text-4xl font-semibold tracking-[-0.035em]">Redirect verified.</p><p class="mt-3 text-[var(--brand-text-secondary)]">Browser returned to /workspace with the session restored.</p><div class="mt-8 flex justify-center gap-8 text-sm"><span>2 files changed</span><span class="text-success">4 checks passed</span></div></div></div>
      </div>
    </article>
  </div>
</section>
