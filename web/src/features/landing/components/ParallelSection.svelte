<script lang="ts">
  const agents = [
    { name: 'Research', state: 'running', detail: 'reading OAuth metadata docs', tone: 'text-primary' },
    { name: 'Code', state: 'review', detail: 'callback.ts · 2 files changed', tone: 'text-secondary' },
    { name: 'Browser', state: 'passed', detail: 'redirect flow verified', tone: 'text-success' },
    { name: 'Verifier', state: 'queued', detail: 'waiting on final diff', tone: 'text-[var(--brand-text-tertiary)]' },
  ] as const;
</script>

<section class="mx-auto w-full max-w-[1440px] px-5 py-24 sm:px-8 lg:px-12 lg:py-32">
  <div class="grid gap-8 lg:grid-cols-[0.72fr_1.28fr] lg:items-end lg:gap-20">
    <h2 class="max-w-xl text-4xl font-semibold tracking-[-0.04em] sm:text-5xl lg:text-6xl">One task. Coordinated agents.</h2>
    <p class="max-w-xl text-base leading-7 text-[var(--brand-text-secondary)] sm:text-lg">The orchestrator routes each part of the job into a visible lane, then brings the evidence back to one task.</p>
  </div>

  <div class="mt-12 overflow-hidden rounded-[1.75rem] border border-[var(--brand-border-strong)] bg-base-100 shadow-[var(--brand-shadow)]">
    <div class="grid grid-cols-3 gap-5 border-b border-[var(--brand-border)] bg-base-200 px-6 py-5 text-sm sm:grid-cols-[1fr_auto_auto]">
      <span>Agent lanes</span><span class="text-[var(--brand-text-tertiary)]">2 active</span><span class="text-[var(--brand-text-tertiary)]">4 total</span>
    </div>
    <div class="divide-y divide-[var(--brand-border)]">
      {#each agents as agent (agent.name)}
        <article class="grid gap-3 px-6 py-5 sm:grid-cols-[1fr_10rem_12rem] sm:items-center">
          <div><h3 class="font-medium">{agent.name}</h3><p class="mt-1 text-sm text-[var(--brand-text-tertiary)]">{agent.detail}</p></div>
          <span class={`text-sm ${agent.tone}`}>{agent.state}</span>
          <div class="h-1.5 overflow-hidden rounded-full bg-base-300"><div class={`h-full rounded-full ${agent.state === 'passed' ? 'w-full bg-success' : agent.state === 'queued' ? 'w-1/5 bg-base-content/20' : 'w-2/3 bg-primary'}`}></div></div>
        </article>
      {/each}
    </div>
  </div>
</section>
