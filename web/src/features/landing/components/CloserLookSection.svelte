<script lang="ts">
  type ViewKey = 'tasks' | 'changes' | 'preview' | 'verify';

  const views: { key: ViewKey; label: string }[] = [
    { key: 'tasks', label: 'Task' },
    { key: 'changes', label: 'Changes' },
    { key: 'preview', label: 'Preview' },
    { key: 'verify', label: 'Verify' },
  ];

  let selected = $state<ViewKey>('tasks');
</script>

<section id="closer-look" class="mx-auto w-full max-w-[1600px] px-5 py-20 sm:px-8 lg:px-12 lg:py-28">
  <div class="flex flex-col gap-8 lg:flex-row lg:items-end lg:justify-between">
    <div>
      <h2 class="max-w-4xl text-4xl font-semibold tracking-[-0.04em] sm:text-5xl lg:text-6xl">The whole task stays in one workspace.</h2>
      <p class="mt-4 max-w-xl text-base leading-7 text-[var(--brand-text-secondary)] sm:text-lg">Switch from progress to code, preview, or verification without losing context.</p>
    </div>
    <div class="max-w-full overflow-x-auto">
      <div class="tabs tabs-border w-max min-w-full sm:min-w-0">
        {#each views as view (view.key)}
          <button class:tab-active={selected === view.key} class="tab" type="button" onclick={() => { selected = view.key; }}>{view.label}</button>
        {/each}
      </div>
    </div>
  </div>

  <div class="mt-10 overflow-hidden rounded-[1.35rem] border border-[var(--brand-border-strong)] bg-base-100">
    <div class="grid min-h-[42rem] lg:grid-cols-[19rem_1fr]">
      <aside class="border-b border-[var(--brand-border)] bg-base-200 p-6 lg:border-b-0 lg:border-r lg:p-7">
        <p class="text-base font-medium">auth redirect</p>
        <p class="mt-1 text-xs text-[var(--brand-text-tertiary)]">main · agent running</p>
        <div class="mt-8 space-y-1 text-sm">
          <div class="rounded-md bg-base-100 px-3 py-2.5">Task</div>
          <div class="px-3 py-2.5 text-[var(--brand-text-secondary)]">AGENTS.md</div>
          <div class="px-3 py-2.5 text-[var(--brand-text-secondary)]">auth/session.ts</div>
          <div class="px-3 py-2.5 text-[var(--brand-text-secondary)]">callback.ts</div>
        </div>
        <div class="mt-8 border-t border-[var(--brand-border)] pt-5 text-xs text-[var(--brand-text-tertiary)]">3 files in context · 2 changed</div>
      </aside>

      <div class="min-w-0 p-6 sm:p-10 lg:p-14">
        {#if selected === 'tasks'}
          <div class="mx-auto max-w-4xl">
            <div class="flex flex-col gap-4 border-b border-[var(--brand-border-strong)] pb-7 sm:flex-row sm:items-end sm:justify-between"><div><p class="text-sm text-[var(--brand-text-tertiary)]">Current task</p><h3 class="mt-2 text-3xl font-semibold tracking-[-0.03em] sm:text-4xl">Fix the sign-in redirect.</h3></div><span class="text-sm text-primary">working</span></div>
            <div class="mt-9 grid gap-0 border-l border-[var(--brand-border-strong)] pl-7">
              <div class="grid gap-1 border-b border-[var(--brand-border)] py-5 sm:grid-cols-[1fr_auto]"><p class="font-medium">Read repository rules</p><p class="text-sm text-success">complete</p></div>
              <div class="grid gap-1 border-b border-[var(--brand-border)] py-5 sm:grid-cols-[1fr_auto]"><p class="font-medium">Trace auth callback</p><p class="text-sm text-success">complete</p></div>
              <div class="grid gap-1 border-b border-[var(--brand-border)] py-5 sm:grid-cols-[1fr_auto]"><p class="font-medium">Apply focused change</p><p class="text-sm text-primary">working</p></div>
              <div class="grid gap-1 py-5 sm:grid-cols-[1fr_auto]"><p class="font-medium text-[var(--brand-text-tertiary)]">Verify in browser</p><p class="text-sm text-[var(--brand-text-tertiary)]">queued</p></div>
            </div>
          </div>
        {:else if selected === 'changes'}
          <div class="overflow-x-auto border border-[var(--brand-border)] font-mono text-sm">
            <div class="min-w-[34rem] border-b border-[var(--brand-border)] bg-base-200 px-5 py-4">auth/session.ts</div>
            <div class="grid min-w-[34rem] grid-cols-[3rem_1fr] gap-x-4 px-5 py-3 text-[var(--brand-text-secondary)]"><span>42</span><code>const next = sanitizeRedirect(requested);</code></div>
            <div class="grid min-w-[34rem] grid-cols-[3rem_1fr] gap-x-4 bg-success/10 px-5 py-3"><span>43</span><code>+ if (!next) return '/';</code></div>
            <div class="grid min-w-[34rem] grid-cols-[3rem_1fr] gap-x-4 bg-success/10 px-5 py-3"><span>44</span><code>+ return completeSignIn(next);</code></div>
            <div class="grid min-w-[34rem] grid-cols-[3rem_1fr] gap-x-4 px-5 py-3 text-[var(--brand-text-secondary)]"><span>45</span><code>}</code></div>
          </div>
        {:else if selected === 'preview'}
          <div class="overflow-hidden border border-[var(--brand-border)]">
            <div class="border-b border-[var(--brand-border)] bg-base-200 px-5 py-4 text-xs text-[var(--brand-text-tertiary)]">127.0.0.1:8080/auth/callback</div>
            <div class="grid min-h-[31rem] place-items-center bg-base-100 p-8 text-center"><div><span class="text-3xl text-success">✓</span><h3 class="mt-5 text-4xl font-semibold tracking-[-0.04em] sm:text-5xl">Signed in.</h3><p class="mt-3 text-[var(--brand-text-secondary)]">Redirect returned to the requested workspace.</p></div></div>
          </div>
        {:else}
          <div class="mx-auto max-w-4xl">
            <div class="flex flex-col gap-4 border-b border-[var(--brand-border-strong)] pb-6 sm:flex-row sm:items-end sm:justify-between"><div><p class="text-sm text-[var(--brand-text-tertiary)]">Verification</p><h3 class="mt-2 text-3xl font-semibold sm:text-4xl">4 checks passed</h3></div><span class="text-success">Ready to ship</span></div>
            <div class="divide-y divide-[var(--brand-border)] text-base"><div class="flex justify-between gap-4 py-6"><span>svelte-check</span><span class="text-success">passed</span></div><div class="flex justify-between gap-4 py-6"><span>eslint</span><span class="text-success">passed</span></div><div class="flex justify-between gap-4 py-6"><span>go test ./...</span><span class="text-success">passed</span></div><div class="flex justify-between gap-4 py-6"><span>browser callback</span><span class="text-success">passed</span></div></div>
          </div>
        {/if}
      </div>
    </div>
  </div>
</section>
