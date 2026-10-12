<script lang="ts">
  type ViewKey = 'tasks' | 'changes' | 'preview' | 'verify';

  const views: { key: ViewKey; label: string }[] = [
    { key: 'tasks', label: 'Tasks' },
    { key: 'changes', label: 'Changes' },
    { key: 'preview', label: 'Preview' },
    { key: 'verify', label: 'Verify' },
  ];

  let selected = $state<ViewKey>('tasks');
</script>

<section id="closer-look" class="mx-auto w-full max-w-[1440px] px-5 py-24 sm:px-8 lg:px-12 lg:py-32">
  <div class="text-center">
    <h2 class="text-4xl font-semibold tracking-[-0.04em] sm:text-5xl lg:text-6xl">Everything stays in the task.</h2>
    <p class="mx-auto mt-4 max-w-xl text-base leading-7 text-[var(--brand-text-secondary)] sm:text-lg">Progress, changes, the running product, and verification are one click apart.</p>
  </div>

  <div class="mt-9 flex justify-center">
    <div class="tabs tabs-box bg-base-100">
      {#each views as view (view.key)}
        <button class:tab-active={selected === view.key} class="tab" type="button" onclick={() => { selected = view.key; }}>{view.label}</button>
      {/each}
    </div>
  </div>

  <div class="mt-8 overflow-hidden rounded-[2rem] border border-[var(--brand-border-strong)] bg-base-100 shadow-[var(--brand-shadow)]">
    <div class="grid min-h-[34rem] lg:grid-cols-[16rem_1fr]">
      <aside class="border-b border-[var(--brand-border)] bg-base-200 p-5 lg:border-b-0 lg:border-r">
        <p class="text-sm font-medium">auth redirect</p>
        <p class="mt-1 text-xs text-[var(--brand-text-tertiary)]">main · agent running</p>
        <div class="mt-7 space-y-2 text-sm">
          <div class="rounded-lg bg-base-100 px-3 py-2">Task</div>
          <div class="px-3 py-2 text-[var(--brand-text-secondary)]">AGENTS.md</div>
          <div class="px-3 py-2 text-[var(--brand-text-secondary)]">auth/session.ts</div>
          <div class="px-3 py-2 text-[var(--brand-text-secondary)]">callback.ts</div>
        </div>
        <div class="mt-8 border-t border-[var(--brand-border)] pt-5 text-xs text-[var(--brand-text-tertiary)]">3 files in context · 2 changed</div>
      </aside>

      <div class="p-5 sm:p-8 lg:p-10">
        {#if selected === 'tasks'}
          <div class="mx-auto max-w-3xl">
            <div class="flex items-center justify-between gap-4"><div><p class="text-sm text-[var(--brand-text-tertiary)]">Current task</p><h3 class="mt-2 text-3xl font-semibold tracking-[-0.03em]">Fix the sign-in redirect.</h3></div><span class="badge badge-primary badge-outline">running</span></div>
            <div class="mt-10 space-y-6 border-l border-[var(--brand-border-strong)] pl-6">
              <div><p class="font-medium">Read repository rules</p><p class="mt-1 text-sm text-success">complete</p></div>
              <div><p class="font-medium">Trace auth callback</p><p class="mt-1 text-sm text-success">complete</p></div>
              <div><p class="font-medium">Apply focused change</p><p class="mt-1 text-sm text-primary">working</p></div>
              <div><p class="font-medium text-[var(--brand-text-tertiary)]">Verify in browser</p><p class="mt-1 text-sm text-[var(--brand-text-tertiary)]">queued</p></div>
            </div>
          </div>
        {:else if selected === 'changes'}
          <div class="overflow-hidden rounded-2xl border border-[var(--brand-border)] font-mono text-sm">
            <div class="border-b border-[var(--brand-border)] bg-base-200 px-5 py-3">auth/session.ts</div>
            <div class="grid grid-cols-[3rem_1fr] gap-x-4 px-5 py-2 text-[var(--brand-text-secondary)]"><span>42</span><code>const next = sanitizeRedirect(requested);</code></div>
            <div class="grid grid-cols-[3rem_1fr] gap-x-4 bg-success/10 px-5 py-2"><span>43</span><code>+ if (!next) return '/';</code></div>
            <div class="grid grid-cols-[3rem_1fr] gap-x-4 bg-success/10 px-5 py-2"><span>44</span><code>+ return completeSignIn(next);</code></div>
            <div class="grid grid-cols-[3rem_1fr] gap-x-4 px-5 py-2 text-[var(--brand-text-secondary)]"><span>45</span><code>}</code></div>
          </div>
        {:else if selected === 'preview'}
          <div class="overflow-hidden rounded-2xl border border-[var(--brand-border)]">
            <div class="flex items-center gap-2 border-b border-[var(--brand-border)] bg-base-200 px-4 py-3 text-xs text-[var(--brand-text-tertiary)]"><span class="size-2 rounded-full bg-error"></span><span class="size-2 rounded-full bg-warning"></span><span class="size-2 rounded-full bg-success"></span><span class="ml-3">127.0.0.1:8080/auth/callback</span></div>
            <div class="grid min-h-[24rem] place-items-center bg-base-100 p-8 text-center"><div><span class="text-success">✓</span><h3 class="mt-4 text-4xl font-semibold tracking-[-0.04em]">Signed in.</h3><p class="mt-3 text-[var(--brand-text-secondary)]">Redirect returned to the requested workspace.</p></div></div>
          </div>
        {:else}
          <div class="mx-auto max-w-3xl">
            <div class="flex items-end justify-between gap-4 border-b border-[var(--brand-border-strong)] pb-5"><div><p class="text-sm text-[var(--brand-text-tertiary)]">Verification</p><h3 class="mt-2 text-3xl font-semibold">4 checks passed</h3></div><span class="text-success">Ready to ship</span></div>
            <div class="divide-y divide-[var(--brand-border)]">
              <div class="flex justify-between py-5"><span>svelte-check</span><span class="text-success">passed</span></div>
              <div class="flex justify-between py-5"><span>eslint</span><span class="text-success">passed</span></div>
              <div class="flex justify-between py-5"><span>go test ./...</span><span class="text-success">passed</span></div>
              <div class="flex justify-between py-5"><span>browser callback</span><span class="text-success">passed</span></div>
            </div>
          </div>
        {/if}
      </div>
    </div>
  </div>
</section>
