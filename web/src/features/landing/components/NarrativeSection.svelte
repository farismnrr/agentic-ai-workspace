<script lang="ts">
  const capabilities = [
    {
      index: '01',
      title: 'Reads before it touches.',
      body: 'Repository rules, architecture, existing abstractions, and current code become context before the first edit happens.',
    },
    {
      index: '02',
      title: 'Operates real tools.',
      body: 'Files, terminal commands, tests, builds, and browser checks live in one execution loop instead of a chain of pasted answers.',
    },
    {
      index: '03',
      title: 'Leaves proof behind.',
      body: 'The result is not “looks right to the model.” The agent runs the project’s verification path and surfaces what actually passed.',
    },
  ] as const;
</script>

<section id="capabilities" class="mx-auto grid w-full max-w-[1440px] gap-16 px-5 py-28 sm:px-8 lg:grid-cols-[0.82fr_1.18fr] lg:px-12 lg:py-40">
  <div>
    <p class="font-mono text-xs font-semibold uppercase tracking-[0.24em] text-primary">System / 01</p>
    <h2 class="mt-6 max-w-xl text-5xl font-semibold leading-[0.94] tracking-[-0.055em] sm:text-6xl lg:text-7xl">
      Not a chatbot.
      <span class="text-[var(--brand-text-tertiary)]">A runtime.</span>
    </h2>
    <p class="mt-7 max-w-md text-base leading-7 text-[var(--brand-text-secondary)]">
      Chat is just the control surface. Under it is an execution loop built to understand a codebase, change it, and prove the change survived reality.
    </p>
  </div>

  <div class="border-t border-[var(--brand-border-strong)]">
    {#each capabilities as item (item.index)}
      <article class="group grid gap-6 border-b border-[var(--brand-border)] py-8 sm:grid-cols-[4rem_1fr] sm:py-10">
        <span class="font-mono text-xs text-[var(--brand-text-tertiary)]">{item.index}</span>
        <div class="grid gap-4 lg:grid-cols-[0.8fr_1.2fr] lg:gap-10">
          <h3 class="text-2xl font-semibold tracking-[-0.035em] transition-transform duration-300 group-hover:translate-x-1">{item.title}</h3>
          <p class="max-w-xl leading-7 text-[var(--brand-text-secondary)]">{item.body}</p>
        </div>
      </article>
    {/each}
  </div>
</section>
