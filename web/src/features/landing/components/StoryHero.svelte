<script lang="ts">
  import { onMount } from 'svelte';

  import LandingNav from './LandingNav.svelte';
  import { createStoryScene, type StorySceneController } from '../visual/storyScene';
  import '../visual/storyWorkspace.css';

  const chapters = [
    {
      eyebrow: 'Agent at work',
      title: 'Give it a goal.',
      body: 'A browser-native coding agent that can read the repo, edit the code, run the tools, and keep going until the result is proven.',
    },
    {
      eyebrow: '01 / Understand',
      title: 'It reads before it writes.',
      body: 'Repository rules, architecture, and relevant files separate into working context before the first change is made.',
    },
    {
      eyebrow: '02 / Execute',
      title: 'It works across the stack.',
      body: 'Editor, terminal, and browser preview become one execution surface instead of a pile of disconnected chat instructions.',
    },
    {
      eyebrow: '03 / Verify',
      title: 'Done means verified.',
      body: 'Tests, lint, builds, and browser checks stay visible, so completion is evidence—not confidence.',
    },
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
    const translateY = (Math.max(-1, Math.min(1, delta)) * 24).toFixed(2);
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

<section bind:this={host} class="relative h-[440svh] bg-base-200 text-base-content">
  <div class="sticky top-0 h-[100svh] overflow-hidden">
    <div bind:this={sceneHost} class="absolute inset-0 z-0" aria-hidden="true"></div>
    <div class="pointer-events-none absolute inset-0 z-10 bg-[radial-gradient(circle_at_72%_44%,var(--brand-ambient-secondary),transparent_28rem)]" aria-hidden="true"></div>

    <div class="relative z-20 mx-auto flex h-full w-full max-w-[1440px] flex-col px-5 sm:px-8 lg:px-12">
      <LandingNav />

      <div class="relative flex-1">
        {#each chapters as chapter, index (chapter.eyebrow)}
          <article
            class="pointer-events-none absolute left-0 top-[48%] max-w-[32rem] will-change-[opacity,transform] sm:top-[46%]"
            style={chapterStyle(index)}
            aria-hidden={activeChapter !== index}
          >
            <p class="font-mono text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-primary">{chapter.eyebrow}</p>
            <h1 class="mt-4 text-5xl font-semibold leading-[0.92] tracking-[-0.055em] sm:text-6xl lg:text-7xl">
              {chapter.title}
            </h1>
            <p class="mt-5 max-w-md text-base leading-7 text-[var(--brand-text-secondary)] sm:text-lg">
              {chapter.body}
            </p>
            {#if index === 0}
              <p class="mt-7 font-mono text-[0.65rem] uppercase tracking-[0.2em] text-[var(--brand-text-tertiary)]">Scroll to follow the agent ↓</p>
            {/if}
          </article>
        {/each}
      </div>

      <div class="absolute right-5 top-1/2 hidden -translate-y-1/2 flex-col gap-3 md:flex lg:right-12" aria-hidden="true">
        {#each chapters as chapter, index (chapter.title)}
          <span class={`h-px w-4 origin-right transition-all duration-300 ${activeChapter === index ? 'scale-x-[2.4] bg-primary' : 'bg-base-content/20'}`}></span>
        {/each}
      </div>
    </div>
  </div>
</section>
