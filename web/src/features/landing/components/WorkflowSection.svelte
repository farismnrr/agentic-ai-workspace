<script lang="ts">
  const steps = [
    { number: '01', title: 'Point at the goal', body: 'Describe the outcome in plain language. Constraints stay attached to the task.' },
    { number: '02', title: 'Watch context assemble', body: 'The agent maps the repo, rules, dependencies, and current documentation before editing.' },
    { number: '03', title: 'Let the loop run', body: 'Search, edit, execute, inspect, and revise happen as one continuous trajectory.' },
    { number: '04', title: 'Review the proof', body: 'You get the change plus the evidence that it actually holds together.' },
  ] as const;
</script>

<section id="how-it-works" class="scroll-mt-28 border-t border-[var(--brand-border)] py-28 lg:py-36">
  <div class="grid gap-14 lg:grid-cols-[0.8fr_1.2fr]">
    <div class="lg:sticky lg:top-28 lg:self-start">
      <p class="text-xs font-semibold uppercase tracking-[0.2em] text-secondary">Agent loop</p>
      <h2 class="mt-4 max-w-md text-4xl font-semibold tracking-[-0.045em] sm:text-5xl">
        Autonomy you can still see.
      </h2>
      <p class="mt-5 max-w-md leading-7 text-[var(--brand-text-secondary)]">
        The interface is designed around progress and evidence instead of a never-ending chat transcript.
      </p>
    </div>

    <div class="divide-y divide-[var(--brand-border)] border-y border-[var(--brand-border)]">
      {#each steps as step}
        <article class="grid gap-4 py-7 sm:grid-cols-[4rem_1fr] sm:py-9">
          <span class="text-sm font-semibold text-primary">{step.number}</span>
          <div>
            <h3 class="text-xl font-semibold tracking-[-0.025em]">{step.title}</h3>
            <p class="mt-2 max-w-xl text-sm leading-6 text-[var(--brand-text-secondary)]">{step.body}</p>
          </div>
        </article>
      {/each}
    </div>
  </div>

  <div id="launch" class="mt-28 overflow-hidden rounded-[2.5rem] border border-[var(--brand-border-strong)] bg-neutral p-7 text-neutral-content sm:p-10 lg:p-14">
    <div class="grid gap-10 lg:grid-cols-[1fr_auto] lg:items-end">
      <div>
        <span class="badge badge-soft rounded-full border-neutral-content/15 bg-neutral-content/10 text-neutral-content">The direction</span>
        <h2 class="mt-6 max-w-3xl text-4xl font-semibold tracking-[-0.05em] sm:text-5xl lg:text-6xl">
          No local ritual. No terminal tax. Just a browser and a goal.
        </h2>
        <p class="mt-5 max-w-xl text-sm leading-6 text-neutral-content/60">
          This first landing pass uses no external illustration assets. The product itself is the visual language.
        </p>
      </div>

      <a class="btn btn-primary rounded-full px-7" href="#workspace-preview">Back to the workspace</a>
    </div>
  </div>
</section>
