<script lang="ts">
  import { onMount } from 'svelte';

  import LandingNav from './LandingNav.svelte';
  import { createStoryScene, type StorySceneController } from '../visual/storyScene';
  import '../visual/storyWorkspace.css';

  const chapters = [
    {
      title: 'Give it a goal.',
      body: 'An agent that works in your codebase, from the browser.',
    },
    { title: 'It reads first.' },
    { title: 'Then it works.' },
    { title: 'Until it passes.' },
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
    const opacity = Math.max(0, 1 - Math.abs(delta)).toFixed(3);
    const translateY = (Math.max(-1, Math.min(1, delta)) * 20).toFixed(2);
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

<section bind:this={host} class="relative h-[400svh] bg-base-200 text-base-content">
  <div class="sticky top-0 h-[100svh] overflow-hidden">
    <div bind:this={sceneHost} class="absolute inset-0 z-0" aria-hidden="true"></div>

    <div class="relative z-20 mx-auto flex h-full w-full max-w-[1440px] flex-col px-5 sm:px-8 lg:px-12">
      <LandingNav />

      <div class="relative flex-1">
        {#each chapters as chapter, index (chapter.title)}
          <article
            class="pointer-events-none absolute left-0 top-[54%] max-w-[28rem] will-change-[opacity,transform] sm:top-[50%]"
            style={chapterStyle(index)}
            aria-hidden={activeChapter !== index}
          >
            <h1 class="text-4xl font-semibold leading-[0.98] tracking-[-0.035em] sm:text-5xl lg:text-6xl">
              {chapter.title}
            </h1>
            {#if 'body' in chapter}
              <p class="mt-5 max-w-sm text-base leading-7 text-[var(--brand-text-secondary)]">
                {chapter.body}
              </p>
              <p class="mt-6 text-sm text-[var(--brand-text-tertiary)]">Scroll to follow the task ↓</p>
            {/if}
          </article>
        {/each}
      </div>
    </div>
  </div>
</section>
