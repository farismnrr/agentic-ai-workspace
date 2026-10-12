<script lang="ts">
  const activity = [
    ['Read official OAuth metadata docs', 'done'],
    ['Inspect protected-resource route', 'done'],
    ['Patch discovery response', 'done'],
    ['Run browser verification', 'running'],
  ] as const;
</script>

<section class="bg-neutral text-neutral-content">
  <div class="mx-auto w-full max-w-[1600px] px-5 py-20 sm:px-8 lg:px-12 lg:py-28">
    <div class="grid gap-10 lg:grid-cols-[0.58fr_1.42fr] lg:items-center lg:gap-16">
      <div>
        <h2 class="max-w-lg text-4xl font-semibold tracking-[-0.04em] sm:text-5xl lg:text-6xl">Start the task. Leave it running.</h2>
        <p class="mt-5 max-w-md text-base leading-7 text-neutral-content/65 sm:text-lg">Context, tools, and progress stay with the task while you step away.</p>
      </div>

      <article class="min-h-[30rem] overflow-hidden border border-neutral-content/15 bg-neutral-content/[0.035]">
        <div class="flex flex-wrap items-end justify-between gap-4 border-b border-neutral-content/10 px-7 py-6 sm:px-9">
          <div><p class="text-sm text-neutral-content/50">farismnrr / agentic-ai-workspace</p><h3 class="mt-2 text-2xl font-semibold sm:text-3xl">Fix OAuth discovery and verify the flow.</h3></div>
          <span class="text-sm text-success">running</span>
        </div>

        <div class="grid min-h-[22rem] md:grid-cols-[1fr_17rem]">
          <div class="px-7 py-6 sm:px-9 sm:py-8">
            {#each activity as item, index (item[0])}
              <div class="grid grid-cols-[1.25rem_1fr_auto] gap-4 border-b border-neutral-content/10 py-5 last:border-b-0">
                <span class={item[1] === 'done' ? 'text-success' : 'text-primary'}>{item[1] === 'done' ? '✓' : '●'}</span>
                <span>{item[0]}</span>
                <span class="text-sm text-neutral-content/45">{index + 1}/4</span>
              </div>
            {/each}
          </div>
          <div class="border-t border-neutral-content/10 bg-neutral-content/[0.025] p-7 md:border-l md:border-t-0 sm:p-9">
            <p class="text-sm text-neutral-content/45">Environment</p><p class="mt-2 font-medium">isolated workspace</p>
            <p class="mt-8 text-sm text-neutral-content/45">Branch</p><p class="mt-2 font-medium">main</p>
            <p class="mt-8 text-sm text-neutral-content/45">Last event</p><p class="mt-2 text-sm leading-6 text-neutral-content/75">Browser opened /oauth-protected-resource.</p>
          </div>
        </div>
      </article>
    </div>
  </div>
</section>
