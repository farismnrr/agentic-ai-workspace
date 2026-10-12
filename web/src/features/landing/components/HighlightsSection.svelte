<section id="highlights" class="bg-base-100">
  <div class="mx-auto w-full max-w-[1440px] px-5 py-24 sm:px-8 lg:px-12 lg:py-32">
    <div class="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
      <h2 class="max-w-3xl text-4xl font-semibold tracking-[-0.04em] sm:text-6xl lg:text-7xl">What happens after you press enter.</h2>
      <p class="max-w-md text-base leading-7 text-[var(--brand-text-secondary)] sm:text-lg">The product is the workflow: context, edits, tools, preview, and proof.</p>
    </div>

    <div class="mt-16 grid gap-5 lg:grid-cols-12 lg:auto-rows-[21rem]">
      <article class="overflow-hidden rounded-3xl border border-[var(--brand-border)] bg-base-200 p-6 sm:p-8 lg:col-span-7">
        <div class="flex items-start justify-between gap-6">
          <div>
            <h3 class="text-2xl font-semibold tracking-[-0.03em] sm:text-3xl">Reads the repo before touching it.</h3>
            <p class="mt-3 max-w-xl leading-7 text-[var(--brand-text-secondary)]">Rules and relevant files become working context, not another thing you have to paste into chat.</p>
          </div>
          <span class="text-sm text-[var(--brand-text-tertiary)]">Context</span>
        </div>
        <div class="mt-8 grid gap-3 sm:grid-cols-[0.8fr_1.2fr]">
          <div class="rounded-2xl border border-[var(--brand-border)] bg-base-100 p-4 font-mono text-xs leading-6">
            <p class="text-primary">AGENTS.md</p>
            <p>web/src/features/</p>
            <p class="pl-4">landing/</p>
            <p class="pl-8 text-base-content">StoryHero.svelte</p>
            <p>internal/modules/</p>
          </div>
          <div class="rounded-2xl border border-[var(--brand-border)] bg-base-100 p-5">
            <p class="text-sm text-[var(--brand-text-tertiary)]">Working context</p>
            <p class="mt-3 font-medium">Read rules before code changes.</p>
            <p class="mt-2 text-sm leading-6 text-[var(--brand-text-secondary)]">Reuse existing UI · preserve light and dark · verify before done.</p>
          </div>
        </div>
      </article>

      <article class="overflow-hidden rounded-3xl bg-neutral p-6 text-neutral-content sm:p-8 lg:col-span-5">
        <h3 class="text-2xl font-semibold tracking-[-0.03em] sm:text-3xl">Runs the tools.</h3>
        <p class="mt-3 max-w-md leading-7 text-neutral-content/70">Commands stay visible while the agent works through failures and retries.</p>
        <div class="mt-8 rounded-2xl border border-neutral-content/15 bg-neutral-content/[0.04] p-5 font-mono text-xs leading-7">
          <p><span class="text-primary">$</span> make verify</p>
          <p class="text-neutral-content/60">checking frontend…</p>
          <p class="text-success">✓ svelte-check</p>
          <p class="text-success">✓ eslint</p>
          <p class="text-success">✓ go test ./...</p>
        </div>
      </article>

      <article class="overflow-hidden rounded-3xl border border-[var(--brand-border)] bg-base-200 p-6 sm:p-8 lg:col-span-5">
        <h3 class="text-2xl font-semibold tracking-[-0.03em] sm:text-3xl">Runs the product too.</h3>
        <p class="mt-3 leading-7 text-[var(--brand-text-secondary)]">A live browser preview sits beside the code instead of becoming a separate manual loop.</p>
        <div class="mt-7 overflow-hidden rounded-2xl border border-[var(--brand-border)] bg-base-100">
          <div class="flex items-center gap-2 border-b border-[var(--brand-border)] px-4 py-3 text-xs text-[var(--brand-text-tertiary)]">
            <span class="size-2 rounded-full bg-error"></span><span class="size-2 rounded-full bg-warning"></span><span class="size-2 rounded-full bg-success"></span>
            <span class="ml-2">127.0.0.1:8080</span>
          </div>
          <div class="grid min-h-36 place-items-center p-6 text-center">
            <div><p class="text-sm text-primary">Masih Awam</p><p class="mt-2 text-2xl font-semibold">Preview ready.</p></div>
          </div>
        </div>
      </article>

      <article class="overflow-hidden rounded-3xl border border-[var(--brand-border)] bg-base-200 p-6 sm:p-8 lg:col-span-7">
        <div class="flex items-start justify-between gap-6">
          <div><h3 class="text-2xl font-semibold tracking-[-0.03em] sm:text-3xl">Leaves proof behind.</h3><p class="mt-3 max-w-xl leading-7 text-[var(--brand-text-secondary)]">You can see what changed, what ran, and why the task is marked done.</p></div>
          <span class="text-sm font-medium text-success">Verified</span>
        </div>
        <dl class="mt-8 divide-y divide-[var(--brand-border)] border-y border-[var(--brand-border)]">
          <div class="grid grid-cols-[7rem_1fr] py-3"><dt class="text-sm text-[var(--brand-text-tertiary)]">Changed</dt><dd class="text-sm">auth/session.ts · callback.ts</dd></div>
          <div class="grid grid-cols-[7rem_1fr] py-3"><dt class="text-sm text-[var(--brand-text-tertiary)]">Ran</dt><dd class="font-mono text-sm">make verify</dd></div>
          <div class="grid grid-cols-[7rem_1fr] py-3"><dt class="text-sm text-[var(--brand-text-tertiary)]">Result</dt><dd class="text-sm text-success">4 checks passed</dd></div>
        </dl>
      </article>
    </div>
  </div>
</section>
