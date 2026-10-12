<script lang="ts">
  import { onMount } from 'svelte';

  import LandingNav from './LandingNav.svelte';
  import { createStoryScene, type StorySceneController } from '../visual/storyScene';
  import '../visual/storyWorkspace.css';
  import '../visual/storyOrchestrator.css';

  const chapters = [
    { title: 'One goal. One orchestrator.', body: 'Masih Awam turns the outcome into a visible plan.' },
    { title: 'It routes the work.', body: 'Research, code, browser, and verification get their own lane.' },
    { title: 'Agents work in the open.', body: 'Context, diffs, commands, and preview stay attached to the task.' },
    { title: 'Proof closes the loop.', body: 'Checks and changed files decide when the work is done.' },
  ] as const;

  let host: HTMLElement;
  let sceneHost: HTMLDivElement;
  let controller: StorySceneController | null = null;
  let progress = $state(0);
  let frame: number | null = null;

  const chapterPosition = $derived(progress * (chapters.length - 1));
  const activeChapter = $derived(Math.min(chapters.length - 1, Math.round(chapterPosition)));

  function chapterStyle(index: number): string {
    const delta = index - chapterPosition;
    const opacity = Math.max(0, 1 - Math.abs(delta) / 0.38).toFixed(3);
    const translateY = (Math.max(-1, Math.min(1, delta)) * 22).toFixed(2);
    return `opacity:${opacity};transform:translate3d(0,${translateY}px,0)`;
  }

  function measure(): void {
    frame = null;
    const rect = host.getBoundingClientRect();
    const travel = Math.max(1, rect.height - window.innerHeight);
    progress = Math.max(0, Math.min(1, -rect.top / travel));
    controller?.progress(progress);
  }

  function queueMeasure(): void {
    if (frame !== null) return;
    frame = window.requestAnimationFrame(measure);
  }

  onMount(() => {
    controller = createStoryScene(sceneHost);
    const resizeObserver = new ResizeObserver(([entry]) => {
      if (entry === undefined) return;
      controller?.resize(entry.contentRect.width, entry.contentRect.height);
      queueMeasure();
    });
    const visibilityObserver = new IntersectionObserver(([entry]) => {
      controller?.active(entry?.isIntersecting === true);
    }, { threshold: 0 });

    resizeObserver.observe(sceneHost);
    visibilityObserver.observe(host);
    controller.resize(sceneHost.clientWidth, sceneHost.clientHeight);
    queueMeasure();

    return () => {
      if (frame !== null) window.cancelAnimationFrame(frame);
      resizeObserver.disconnect();
      visibilityObserver.disconnect();
      controller?.destroy();
      controller = null;
    };
  });
</script>

<svelte:window onscroll={queueMeasure} onresize={queueMeasure} />

<section bind:this={host} class="relative h-[360svh] bg-base-200 text-base-content">
  <div class="sticky top-0 h-[100svh] overflow-hidden">
    <div bind:this={sceneHost} class="absolute inset-x-0 bottom-0 top-[30%] z-0 overflow-hidden sm:top-[28%]" aria-hidden="true"></div>

    <div class="relative z-20 mx-auto flex h-full w-full max-w-[1440px] flex-col px-5 sm:px-8 lg:px-12">
      <LandingNav />

      <div class="relative flex-1">
        {#each chapters as chapter, index (chapter.title)}
          <article
            class={`pointer-events-none absolute will-change-[opacity,transform] ${index === 0 ? 'left-1/2 top-[5%] w-[min(92vw,64rem)]' : 'left-0 top-[24%] max-w-[24rem]'}`}
            style={chapterStyle(index)}
            aria-hidden={activeChapter !== index}
          >
            <div class={index === 0 ? '-translate-x-1/2 text-center' : ''}>
              <h1 class={`${index === 0 ? 'text-5xl sm:text-6xl lg:text-7xl' : 'text-4xl sm:text-5xl lg:text-6xl'} font-semibold leading-[0.96] tracking-[-0.05em]`}>
                {chapter.title}
              </h1>
              <p class={`mt-4 text-base leading-7 text-[var(--brand-text-secondary)] sm:text-lg ${index === 0 ? 'mx-auto max-w-lg' : 'max-w-sm'}`}>
                {chapter.body}
              </p>
              {#if index === 0}
                <div class="pointer-events-auto mt-6 flex flex-wrap justify-center gap-2">
                  <a class="btn btn-primary" href="#highlights">See orchestration</a>
                  <a class="btn btn-ghost" href="#launch">Open workspace</a>
                </div>
              {/if}
            </div>
          </article>
        {/each}
      </div>
    </div>
  </div>
</section>
