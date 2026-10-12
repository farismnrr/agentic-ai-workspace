<script lang="ts">
  const trace = [
    { label: 'Read', value: 'AGENTS.md · auth/session.ts · callback.ts' },
    { label: 'Changed', value: '2 files' },
    { label: 'Ran', value: 'make verify' },
    { label: 'Result', value: '4 checks passed' },
  ] as const;
</script>

<section id="how-it-works" class="mx-auto w-full max-w-[1440px] px-5 pb-28 sm:px-8 lg:px-12 lg:pb-40">
  <div class="grid gap-10 border-t border-[var(--brand-border-strong)] pt-16 lg:grid-cols-[0.72fr_1.28fr] lg:gap-20 lg:pt-20">
    <div>
      <h2 class="max-w-md text-4xl font-semibold leading-tight tracking-[-0.035em] sm:text-5xl lg:text-6xl">
        See what it did.
      </h2>
      <p class="mt-6 max-w-md leading-7 text-[var(--brand-text-secondary)]">
        The task, edits, commands, and proof stay visible instead of disappearing behind a confident answer.
      </p>
    </div>

    <div class="overflow-hidden rounded-2xl border border-[var(--brand-border-strong)] bg-base-100">
      <div class="flex items-center justify-between gap-6 border-b border-[var(--brand-border)] px-5 py-4 sm:px-6">
        <div>
          <p class="text-sm text-[var(--brand-text-tertiary)]">Task</p>
          <p class="mt-1 font-medium">Fix the sign-in redirect</p>
        </div>
        <span class="text-sm font-medium text-success">Verified</span>
      </div>

      <dl class="divide-y divide-[var(--brand-border)]">
        {#each trace as item (item.label)}
          <div class="grid gap-2 px-5 py-4 sm:grid-cols-[7rem_1fr] sm:px-6">
            <dt class="text-sm text-[var(--brand-text-tertiary)]">{item.label}</dt>
            <dd class="text-sm sm:text-base">{item.value}</dd>
          </div>
        {/each}
      </dl>

      <div class="flex flex-wrap items-center justify-between gap-3 border-t border-[var(--brand-border)] bg-base-200 px-5 py-4 font-mono text-xs sm:px-6">
        <span class="text-[var(--brand-text-secondary)]">$ make verify</span>
        <span class="text-success">frontend · tests · build · browser ✓</span>
      </div>
    </div>
  </div>
</section>
