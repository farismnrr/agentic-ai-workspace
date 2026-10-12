<script lang="ts">
  const activity = [
    ['Read official OAuth metadata docs', 'done'],
    ['Inspect protected-resource route', 'done'],
    ['Patch discovery response', 'done'],
    ['Run browser verification', 'running'],
  ] as const;
</script>

<section class="bg-neutral text-neutral-content">
  <div class="mx-auto w-full max-w-[1440px] px-5 py-24 sm:px-8 lg:px-12 lg:py-36">
    <div class="grid gap-14 lg:grid-cols-[0.72fr_1.28fr] lg:items-center lg:gap-24">
      <div>
        <h2 class="max-w-lg text-4xl font-semibold tracking-[-0.04em] sm:text-5xl lg:text-6xl">Start the task. Leave it running.</h2>
        <p class="mt-6 max-w-md text-base leading-7 text-neutral-content/65 sm:text-lg">The workspace keeps its context, tools, and progress while you do something else.</p>
      </div>

      <article class="overflow-hidden rounded-[2rem] border border-neutral-content/15 bg-neutral-content/[0.035]">
        <div class="flex flex-wrap items-center justify-between gap-4 border-b border-neutral-content/10 px-6 py-5 sm:px-8">
          <div><p class="text-sm text-neutral-content/50">farismnrr / agentic-ai-workspace</p><h3 class="mt-2 text-xl font-semibold sm:text-2xl">Fix OAuth discovery and verify the flow.</h3></div>
          <span class="badge badge-success badge-outline">running</span>
        </div>

        <div class="grid md:grid-cols-[1fr_14rem]">
          <div class="px-6 py-7 sm:px-8">
            <div class="space-y-0">
              {#each activity as item, index (item[0])}
                <div class="grid grid-cols-[1.25rem_1fr_auto] gap-4 border-b border-neutral-content/10 py-4 last:border-b-0">
                  <span class={item[1] === 'done' ? 'text-success' : 'text-primary'}>{item[1] === 'done' ? '✓' : '●'}</span>
                  <span>{item[0]}</span>
                  <span class="text-sm text-neutral-content/45">{index + 1}/4</span>
                </div>
              {/each}
            </div>
          </div>
          <div class="border-t border-neutral-content/10 bg-neutral-content/[0.025] p-6 md:border-l md:border-t-0 sm:p-8">
            <p class="text-sm text-neutral-content/45">Environment</p><p class="mt-2 font-medium">isolated workspace</p>
            <p class="mt-8 text-sm text-neutral-content/45">Branch</p><p class="mt-2 font-medium">main</p>
            <p class="mt-8 text-sm text-neutral-content/45">Last event</p><p class="mt-2 text-sm leading-6 text-neutral-content/75">Browser opened /oauth-protected-resource.</p>
          </div>
        </div>
      </article>
    </div>
  </div>
</section>
