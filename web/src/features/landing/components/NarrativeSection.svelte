<script lang="ts">
  const principles = [
    {
      index: '01',
      title: 'Context before confidence.',
      body: 'The agent reads repository rules, architecture, and existing patterns before it proposes a change.',
    },
    {
      index: '02',
      title: 'Action over suggestions.',
      body: 'It edits files, runs commands, and works through the same project surface a developer would use.',
    },
    {
      index: '03',
      title: 'Evidence before done.',
      body: 'The loop ends with tests, lint, builds, and browser checks—not with a persuasive paragraph.',
    },
  ] as const;
</script>

<section id="capabilities" class="mx-auto w-full max-w-[1440px] px-5 py-28 sm:px-8 lg:px-12 lg:py-44">
  <div class="max-w-5xl">
    <p class="font-mono text-xs font-semibold uppercase tracking-[0.24em] text-primary">After the prompt</p>
    <h2 class="mt-6 text-5xl font-semibold leading-[0.94] tracking-[-0.055em] sm:text-7xl lg:text-8xl">
      The interface gets quiet.
      <span class="block text-[var(--brand-text-tertiary)]">The work gets serious.</span>
    </h2>
    <p class="mt-8 max-w-2xl text-lg leading-8 text-[var(--brand-text-secondary)]">
      Masih Awam is built around an execution loop, not a conversation loop. The browser is simply where that work becomes visible.
    </p>
  </div>

  <div class="mt-24 border-t border-[var(--brand-border-strong)] lg:mt-32">
    {#each principles as item (item.index)}
      <article class="grid gap-6 border-b border-[var(--brand-border)] py-9 sm:grid-cols-[5rem_1fr] lg:grid-cols-[6rem_0.8fr_1.2fr] lg:items-start lg:py-12">
        <span class="font-mono text-xs text-[var(--brand-text-tertiary)]">{item.index}</span>
        <h3 class="text-2xl font-semibold tracking-[-0.035em] sm:text-3xl">{item.title}</h3>
        <p class="max-w-xl text-base leading-7 text-[var(--brand-text-secondary)] sm:text-lg sm:leading-8">{item.body}</p>
      </article>
    {/each}
  </div>
</section>
