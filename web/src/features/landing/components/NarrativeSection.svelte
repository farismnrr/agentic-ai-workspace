<script lang="ts">
  const steps = [
    {
      title: 'Read',
      body: 'Rules, architecture, and the files that matter come first.',
    },
    {
      title: 'Change',
      body: 'Edits happen in the repo, not in a chat transcript.',
    },
    {
      title: 'Verify',
      body: 'Commands, tests, and browser checks decide when the task is done.',
    },
  ] as const;
</script>

<section id="capabilities" class="mx-auto w-full max-w-[1440px] px-5 py-24 sm:px-8 lg:px-12 lg:py-32">
  <div class="grid gap-8 lg:grid-cols-[0.85fr_1.15fr] lg:gap-20">
    <h2 class="max-w-xl text-4xl font-semibold leading-tight tracking-[-0.035em] sm:text-5xl lg:text-6xl">
      From task to verified change.
    </h2>
    <p class="max-w-xl text-base leading-7 text-[var(--brand-text-secondary)] lg:pt-3 lg:text-lg lg:leading-8">
      The agent reads the repo, makes the change, and runs the checks in the same workspace.
    </p>
  </div>

  <div class="mt-16 border-t border-[var(--brand-border-strong)] lg:mt-20">
    {#each steps as step (step.title)}
      <article class="grid gap-3 border-b border-[var(--brand-border)] py-7 sm:grid-cols-[10rem_1fr] sm:gap-10 lg:py-9">
        <h3 class="text-lg font-medium">{step.title}</h3>
        <p class="max-w-2xl leading-7 text-[var(--brand-text-secondary)]">{step.body}</p>
      </article>
    {/each}
  </div>
</section>
