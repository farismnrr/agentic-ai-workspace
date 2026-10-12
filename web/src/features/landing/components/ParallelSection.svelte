<script lang="ts">
  const tasks = [
    { name: 'Auth redirect', state: 'running', detail: 'browser verification', tone: 'text-primary' },
    { name: 'Landing polish', state: 'review', detail: '3 files changed', tone: 'text-secondary' },
    { name: 'Dependency update', state: 'passed', detail: '18 checks', tone: 'text-success' },
    { name: 'API refactor', state: 'queued', detail: 'waiting for worker', tone: 'text-[var(--brand-text-tertiary)]' },
  ] as const;
</script>

<section class="mx-auto w-full max-w-[1440px] px-5 py-24 sm:px-8 lg:px-12 lg:py-32">
  <div class="grid gap-8 lg:grid-cols-[0.72fr_1.28fr] lg:items-end lg:gap-20">
    <h2 class="max-w-xl text-4xl font-semibold tracking-[-0.04em] sm:text-5xl lg:text-6xl">Run more than one task.</h2>
    <p class="max-w-xl text-base leading-7 text-[var(--brand-text-secondary)] sm:text-lg">Each task keeps its own context, progress, changes, and result. Switch without losing the thread.</p>
  </div>

  <div class="mt-12 overflow-hidden rounded-[1.75rem] border border-[var(--brand-border-strong)] bg-base-100 shadow-[var(--brand-shadow)]">
    <div class="grid grid-cols-3 gap-5 border-b border-[var(--brand-border)] bg-base-200 px-6 py-5 text-sm sm:grid-cols-[1fr_auto_auto]">
      <span>Tasks</span><span class="text-[var(--brand-text-tertiary)]">2 active</span><span class="text-[var(--brand-text-tertiary)]">4 total</span>
    </div>
    <div class="divide-y divide-[var(--brand-border)]">
      {#each tasks as task (task.name)}
        <article class="grid gap-3 px-6 py-5 sm:grid-cols-[1fr_10rem_12rem] sm:items-center">
          <div><h3 class="font-medium">{task.name}</h3><p class="mt-1 text-sm text-[var(--brand-text-tertiary)]">{task.detail}</p></div>
          <span class={`text-sm ${task.tone}`}>{task.state}</span>
          <div class="h-1.5 overflow-hidden rounded-full bg-base-300"><div class={`h-full rounded-full ${task.state === 'passed' ? 'w-full bg-success' : task.state === 'queued' ? 'w-1/5 bg-base-content/20' : 'w-2/3 bg-primary'}`}></div></div>
        </article>
      {/each}
    </div>
  </div>
</section>
