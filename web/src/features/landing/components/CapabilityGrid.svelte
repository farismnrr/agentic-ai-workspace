<script lang="ts">
  const capabilities = [
    {
      eyebrow: 'Context first',
      title: 'It reads before it writes.',
      body: 'Repository rules, architecture, existing patterns, and current docs become part of the task—not an afterthought.',
      role: 'primary',
    },
    {
      eyebrow: 'Real execution',
      title: 'Not a code vending machine.',
      body: 'The agent can move through files, commands, browser feedback, and verification as one connected work loop.',
      role: 'secondary',
    },
    {
      eyebrow: 'Visible proof',
      title: 'Ship evidence, not vibes.',
      body: 'Diffs, checks, logs, and outcomes stay visible so autonomy never means mystery.',
      role: 'accent',
    },
  ] as const;
</script>

<section id="capabilities" class="scroll-mt-28 py-28 lg:py-36">
  <div class="grid gap-10 lg:grid-cols-[0.72fr_1.28fr] lg:items-end">
    <div>
      <p class="text-xs font-semibold uppercase tracking-[0.2em] text-primary">Built for delegation</p>
      <h2 class="mt-4 text-4xl font-semibold tracking-[-0.045em] sm:text-5xl">
        The browser becomes the engineering surface.
      </h2>
    </div>
    <p class="max-w-2xl text-base leading-7 text-[var(--brand-text-secondary)] lg:justify-self-end">
      The point is not to imitate an IDE pixel-for-pixel. It is to make giving work to an agent feel natural:
      describe the outcome, keep the reasoning inspectable, and stay close to the proof.
    </p>
  </div>

  <div class="mt-12 grid gap-4 lg:grid-cols-3">
    {#each capabilities as item, index (item.title)}
      <article
        class="group relative min-h-80 overflow-hidden rounded-[2rem] border border-[var(--brand-border)] bg-base-100 p-7 transition duration-300 hover:-translate-y-1 hover:border-[var(--brand-border-strong)]"
      >
        <div
          class:bg-primary={item.role === 'primary'}
          class:bg-secondary={item.role === 'secondary'}
          class:bg-accent={item.role === 'accent'}
          class="absolute -right-12 -top-12 size-36 rounded-full opacity-[0.08] blur-2xl transition-transform duration-500 group-hover:scale-150"
          aria-hidden="true"
        ></div>

        <div class="flex items-center justify-between">
          <span class="text-xs font-semibold uppercase tracking-[0.18em] text-[var(--brand-text-tertiary)]">{item.eyebrow}</span>
          <span class="text-xs text-[var(--brand-text-tertiary)]">0{index + 1}</span>
        </div>

        <div class="mt-24">
          <h3 class="text-2xl font-semibold tracking-[-0.035em]">{item.title}</h3>
          <p class="mt-4 text-sm leading-6 text-[var(--brand-text-secondary)]">{item.body}</p>
        </div>
      </article>
    {/each}
  </div>
</section>
