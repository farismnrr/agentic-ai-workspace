<script lang="ts">
</script>

<section id="highlights" class="bg-base-100">
  <div class="mx-auto w-full max-w-[1440px] px-5 py-24 sm:px-8 lg:px-12 lg:py-32">
    <div class="max-w-3xl">
      <h2 class="text-4xl font-semibold tracking-[-0.04em] sm:text-5xl lg:text-6xl">Orchestration you can inspect.</h2>
      <p class="mt-5 max-w-xl text-base leading-7 text-[var(--brand-text-secondary)] sm:text-lg">One goal becomes context, agent lanes, execution, preview, and proof without disappearing into a black box.</p>
    </div>

    <article class="mt-16 grid gap-10 border-t border-[var(--brand-border-strong)] pt-12 lg:grid-cols-[0.7fr_1.3fr] lg:items-center lg:gap-20">
      <div>
        <p class="text-sm font-medium text-primary">Plan</p>
        <h3 class="mt-3 text-3xl font-semibold tracking-[-0.035em] sm:text-4xl">It reads before it routes.</h3>
        <p class="mt-4 max-w-md leading-7 text-[var(--brand-text-secondary)]">Rules, architecture, and relevant files become context before work is handed to an agent lane.</p>
      </div>
      <div class="overflow-hidden rounded-[1.75rem] border border-[var(--brand-border)] bg-base-200 shadow-[var(--brand-shadow)]">
        <div class="grid min-h-80 sm:grid-cols-[0.8fr_1.2fr]">
          <div class="border-b border-[var(--brand-border)] p-6 font-mono text-xs leading-7 sm:border-b-0 sm:border-r">
            <p class="font-sans text-sm font-medium text-base-content">Repository</p>
            <p class="mt-5 text-primary">AGENTS.md</p><p>web/</p><p class="pl-4">src/</p><p class="pl-8">features/auth/</p><p class="pl-8 text-base-content">callback.ts</p><p class="pl-8">session.ts</p><p>tests/</p>
          </div>
          <div class="grid content-center p-7 sm:p-10">
            <p class="text-sm text-[var(--brand-text-tertiary)]">Working context</p>
            <p class="mt-4 max-w-lg text-2xl font-medium tracking-[-0.02em]">Read repository rules before changing code.</p>
            <div class="mt-8 space-y-3 text-sm text-[var(--brand-text-secondary)]"><p>✓ reuse existing UI</p><p>✓ preserve light and dark</p><p>✓ verify before done</p></div>
          </div>
        </div>
      </div>
    </article>

    <article class="mt-20 rounded-[2rem] bg-neutral p-6 text-neutral-content sm:p-10 lg:p-12">
      <div class="grid gap-10 lg:grid-cols-[0.6fr_1.4fr] lg:items-center lg:gap-16">
        <div>
          <p class="text-sm font-medium text-primary">Execution</p>
          <h3 class="mt-3 text-3xl font-semibold tracking-[-0.035em] sm:text-4xl">The work stays visible.</h3>
          <p class="mt-4 max-w-sm leading-7 text-neutral-content/65">Edits, commands, and the running product stay side by side while the task moves.</p>
        </div>
        <div class="grid gap-4 md:grid-cols-2">
          <div class="overflow-hidden rounded-2xl border border-neutral-content/15 bg-neutral-content/[0.04]">
            <div class="border-b border-neutral-content/10 px-5 py-3 font-mono text-xs text-neutral-content/55">callback.ts · modified</div>
            <div class="overflow-x-auto p-5 font-mono text-xs leading-7"><p class="whitespace-nowrap">41&nbsp; const next = params.get('next');</p><p class="whitespace-nowrap bg-success/10 text-success">42&nbsp; + const target = safeRedirect(next);</p><p class="whitespace-nowrap bg-success/10 text-success">43&nbsp; + return redirect(target);</p></div>
          </div>
          <div class="overflow-hidden rounded-2xl border border-neutral-content/15 bg-neutral-content/[0.04]">
            <div class="border-b border-neutral-content/10 px-5 py-3 font-mono text-xs text-neutral-content/55">Terminal</div>
            <div class="overflow-x-auto p-5 font-mono text-xs leading-7"><p class="whitespace-nowrap"><span class="text-primary">$</span> make verify</p><p class="whitespace-nowrap text-success">✓ frontend · passed</p><p class="whitespace-nowrap text-success">✓ tests · passed</p><p class="whitespace-nowrap text-success">✓ production build · passed</p></div>
          </div>
        </div>
      </div>
    </article>

    <article class="mt-20 grid gap-10 border-t border-[var(--brand-border-strong)] pt-12 lg:grid-cols-[1.3fr_0.7fr] lg:items-center lg:gap-20">
      <div class="overflow-hidden rounded-[1.75rem] border border-[var(--brand-border)] bg-base-200 shadow-[var(--brand-shadow)]">
        <div class="flex items-center gap-2 border-b border-[var(--brand-border)] px-5 py-4 text-xs text-[var(--brand-text-tertiary)]"><span class="size-2 rounded-full bg-error"></span><span class="size-2 rounded-full bg-warning"></span><span class="size-2 rounded-full bg-success"></span><span class="ml-2">127.0.0.1:8080/auth/callback</span></div>
        <div class="grid min-h-80 place-items-center p-8 text-center"><div><span class="text-3xl text-success">✓</span><p class="mt-4 text-3xl font-semibold tracking-[-0.03em]">Redirect verified.</p><p class="mt-3 text-[var(--brand-text-secondary)]">Browser check returned to /workspace.</p><div class="mt-7 flex justify-center gap-6 text-sm"><span>2 files changed</span><span class="text-success">4 checks passed</span></div></div></div>
      </div>
      <div>
        <p class="text-sm font-medium text-success">Proof</p>
        <h3 class="mt-3 text-3xl font-semibold tracking-[-0.035em] sm:text-4xl">Done means checked.</h3>
        <p class="mt-4 max-w-md leading-7 text-[var(--brand-text-secondary)]">The result ends with a diff, a running product, and checks you can inspect.</p>
      </div>
    </article>
  </div>
</section>
