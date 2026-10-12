<script lang="ts">
  const trace = [
    { label: 'Reading repository rules', meta: 'AGENTS.md', state: 'done' },
    { label: 'Mapping the codebase', meta: '42 files', state: 'done' },
    { label: 'Editing landing experience', meta: '6 files', state: 'active' },
    { label: 'Running verification', meta: 'queued', state: 'queued' },
  ] as const;
</script>

<div id="workspace-preview" class="relative mx-auto w-full max-w-[720px] scroll-mt-28">
  <div
    class="absolute inset-x-[8%] top-[8%] h-[78%] rounded-[4rem] bg-secondary/10 blur-3xl"
    aria-hidden="true"
  ></div>
  <div class="stage-grid absolute inset-0 opacity-45" aria-hidden="true"></div>

  <div class="relative py-8 sm:px-7 sm:py-12">
    <div class="stage-shell relative mx-auto max-w-[620px]">
      <div class="mb-3 flex items-center justify-between px-2 text-xs text-[var(--brand-text-tertiary)]">
        <span>workspace / launch-page</span>
        <span class="flex items-center gap-2">
          <span class="size-1.5 rounded-full bg-success"></span>
          agent online
        </span>
      </div>

      <div class="rounded-[2rem] border border-[var(--brand-border-strong)] bg-base-100/90 p-3 shadow-[var(--brand-shadow)] backdrop-blur-xl">
        <div class="rounded-[1.45rem] border border-[var(--brand-border)] bg-base-200 p-4 sm:p-5">
          <div class="flex items-start justify-between gap-4">
            <div>
              <p class="text-xs font-semibold uppercase tracking-[0.16em] text-primary">Current task</p>
              <p class="mt-2 max-w-md text-sm font-medium leading-6 sm:text-base">
                “Make the landing page feel unmistakably agentic, spatial, and alive.”
              </p>
            </div>
            <span class="badge badge-soft badge-secondary hidden sm:inline-flex">working</span>
          </div>
        </div>

        <div class="mt-3 grid gap-3 sm:grid-cols-[1fr_0.72fr]">
          <div class="rounded-[1.45rem] border border-[var(--brand-border)] bg-base-200 p-4">
            <div class="mb-4 flex items-center justify-between">
              <span class="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--brand-text-tertiary)]">Agent trace</span>
              <span class="loading loading-dots loading-xs text-primary" aria-label="Agent is working"></span>
            </div>

            <div class="space-y-1">
              {#each trace as item (item.label)}
                <div class="flex items-center gap-3 rounded-xl px-2 py-2.5">
                  <span
                    class:animate-pulse={item.state === 'active'}
                    class="grid size-5 shrink-0 place-items-center rounded-full border border-[var(--brand-border-strong)] text-[0.6rem]"
                  >
                    {item.state === 'done' ? '✓' : item.state === 'active' ? '•' : '·'}
                  </span>
                  <span class="min-w-0 flex-1 truncate text-sm">{item.label}</span>
                  <span class="shrink-0 text-xs text-[var(--brand-text-tertiary)]">{item.meta}</span>
                </div>
              {/each}
            </div>
          </div>

          <div class="grid gap-3">
            <div class="rounded-[1.45rem] border border-[var(--brand-border)] bg-neutral p-4 text-neutral-content">
              <div class="flex items-center gap-1.5" aria-hidden="true">
                <span class="size-2 rounded-full bg-error"></span>
                <span class="size-2 rounded-full bg-warning"></span>
                <span class="size-2 rounded-full bg-success"></span>
              </div>
              <pre class="mt-5 overflow-hidden text-[0.69rem] leading-5 text-neutral-content/70"><code>$ make verify
✓ svelte-check
✓ eslint
… building app</code></pre>
            </div>

            <div class="rounded-[1.45rem] border border-[var(--brand-border)] bg-primary p-4 text-primary-content">
              <p class="text-xs font-semibold uppercase tracking-[0.16em] opacity-65">Proof</p>
              <p class="mt-2 text-3xl font-semibold tracking-[-0.05em]">3 / 4</p>
              <p class="mt-1 text-xs opacity-70">quality gates complete</p>
            </div>
          </div>
        </div>
      </div>

      <div class="float-card absolute -left-2 top-[40%] hidden rounded-2xl border border-[var(--brand-border)] bg-base-100/90 px-3 py-2 text-xs shadow-lg backdrop-blur md:block">
        <span class="text-[var(--brand-text-tertiary)]">context</span>
        <strong class="ml-2">86%</strong>
      </div>

      <div class="float-card float-card-delay absolute -right-3 bottom-[18%] hidden rounded-2xl border border-[var(--brand-border)] bg-base-100/90 px-3 py-2 text-xs shadow-lg backdrop-blur md:block">
        <span class="text-success">●</span>
        <strong class="ml-2">checks passing</strong>
      </div>
    </div>
  </div>
</div>

<style>
  .stage-grid {
    background-image:
      linear-gradient(var(--brand-border) 1px, transparent 1px),
      linear-gradient(90deg, var(--brand-border) 1px, transparent 1px);
    background-size: 34px 34px;
    mask-image: radial-gradient(circle at center, black 18%, transparent 72%);
  }

  .stage-shell {
    transform: perspective(1200px) rotateY(-4deg) rotateX(2deg);
    transform-style: preserve-3d;
  }

  .float-card {
    animation: float-card 5.5s var(--brand-ease) infinite;
  }

  .float-card-delay {
    animation-delay: -2.4s;
  }

  @keyframes float-card {
    0%,
    100% {
      transform: translate3d(0, -4px, 28px);
    }

    50% {
      transform: translate3d(0, 7px, 38px);
    }
  }
</style>
