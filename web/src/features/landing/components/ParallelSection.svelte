<script lang="ts">
  const agents = [
    { name: 'Research', state: 'running', detail: 'OAuth metadata · official docs', tone: 'text-primary' },
    { name: 'Code', state: 'review', detail: 'callback.ts · 2 files changed', tone: 'text-secondary' },
    { name: 'Browser', state: 'passed', detail: 'redirect flow · /workspace', tone: 'text-success' },
    { name: 'Verifier', state: 'queued', detail: 'final diff · waiting', tone: 'text-[var(--brand-text-tertiary)]' },
  ] as const;
</script>

<section class="mx-auto w-full max-w-[1600px] px-5 py-20 sm:px-8 lg:px-12 lg:py-28">
  <div class="grid gap-6 lg:grid-cols-[0.72fr_1.28fr] lg:items-end lg:gap-16">
    <h2 class="max-w-xl text-4xl font-semibold tracking-[-0.04em] sm:text-5xl lg:text-6xl">One task. Coordinated agents.</h2>
    <p class="max-w-xl text-base leading-7 text-[var(--brand-text-secondary)] sm:text-lg">The orchestrator routes work into visible lanes and brings every result back to the same task.</p>
  </div>

  <div class="mt-10 overflow-hidden border-y border-[var(--brand-border-strong)] bg-base-100">
    <div class="hidden grid-cols-[0.7fr_1.6fr_0.5fr] gap-6 border-b border-[var(--brand-border)] px-6 py-4 text-sm text-[var(--brand-text-tertiary)] sm:grid lg:px-8">
      <span>Agent</span><span>Current work</span><span>Status</span>
    </div>
    <div class="divide-y divide-[var(--brand-border)]">
      {#each agents as agent (agent.name)}
        <article class="grid gap-2 px-6 py-6 sm:grid-cols-[0.7fr_1.6fr_0.5fr] sm:items-center sm:gap-6 lg:px-8 lg:py-7">
          <h3 class="text-lg font-medium">{agent.name}</h3>
          <p class="text-sm text-[var(--brand-text-secondary)] sm:text-base">{agent.detail}</p>
          <span class={`text-sm ${agent.tone}`}>{agent.state}</span>
        </article>
      {/each}
    </div>
  </div>
</section>
