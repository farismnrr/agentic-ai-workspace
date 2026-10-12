<script lang="ts">
  import { onMount } from 'svelte';

  import LandingNav from './LandingNav.svelte';
  import { createStoryScene, type StorySceneController } from '../visual/storyScene';

  const chapters = [
    {
      eyebrow: 'Masih Awam',
      title: 'Build it. From here.',
      body: 'Give one goal to an agentic workspace that can understand the repo, operate the tools, and stay until the result is proven.',
    },
    {
      eyebrow: '01 / Understand',
      title: 'First, it understands.',
      body: 'Repository rules, architecture, existing abstractions, and relevant files become working context before the first edit.',
    },
    {
      eyebrow: '02 / Act',
      title: 'Then it gets to work.',
      body: 'The workspace opens the code, changes the smallest surface, runs commands, and keeps the execution visible while it moves.',
    },
    {
      eyebrow: '03 / Verify',
      title: 'Not done until it works.',
      body: 'Tests, lint, build, and browser checks turn an AI answer into evidence you can actually ship.',
    },
  ] as const;

  let host: HTMLElement;
  let canvas: HTMLCanvasElement;
  let controller: StorySceneController | null = null;
  let progress = $state(0);
  let supported = $state(true);
  let frame: number | null = null;

  const chapterPosition = $derived(progress * (chapters.length - 1));
  const activeChapter = $derived(Math.min(chapters.length - 1, Math.round(chapterPosition)));

  function chapterStyle(index: number): string {
    const delta = index - chapterPosition;
    const opacity = Math.max(0, 1 - Math.abs(delta)).toFixed(3);
    const translateY = (Math.max(-1, Math.min(1, delta)) * 28).toFixed(2);
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
    controller = createStoryScene(canvas);
    supported = controller !== null;
    const canvasHost = canvas.parentElement ?? host;
    const resizeObserver = new ResizeObserver(([entry]) => {
      if (entry === undefined) return;
      controller?.resize(entry.contentRect.width, entry.contentRect.height);
      queueMeasure();
    });
    const themeObserver = new MutationObserver(() => controller?.refreshTheme());
    const visibilityObserver = new IntersectionObserver(([entry]) => {
      controller?.active(entry?.isIntersecting === true);
    }, { threshold: 0 });

    resizeObserver.observe(canvasHost);
    visibilityObserver.observe(host);
    themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] });
    controller?.resize(canvasHost.clientWidth, canvasHost.clientHeight);
    queueMeasure();

    return () => {
      if (frame !== null) window.cancelAnimationFrame(frame);
      resizeObserver.disconnect();
      visibilityObserver.disconnect();
      themeObserver.disconnect();
      controller?.destroy();
      controller = null;
    };
  });
</script>

<svelte:window onscroll={queueMeasure} onresize={queueMeasure} />

<section bind:this={host} class="relative h-[420svh] bg-base-200 text-base-content">
  <div class="sticky top-0 h-[100svh] overflow-hidden will-change-transform">
    <div class="absolute inset-0">
      <canvas bind:this={canvas} class="size-full" aria-hidden="true"></canvas>
      {#if !supported}
        <div class="absolute left-1/2 top-1/2 h-[38vh] w-[58vw] max-w-3xl -translate-x-1/2 -translate-y-1/2 rounded-[2.5rem] border border-base-content/15 bg-base-100 shadow-[var(--brand-shadow)]" aria-hidden="true"></div>
      {/if}
    </div>

    <div class="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_center,transparent_28%,var(--color-base-200)_88%)]" aria-hidden="true"></div>

    <div class="relative z-20 mx-auto flex h-full w-full max-w-[1440px] flex-col px-5 sm:px-8 lg:px-12">
      <LandingNav />

      <div class="relative flex-1">
        {#each chapters as chapter, index (chapter.eyebrow)}
          <article
            class="pointer-events-none absolute inset-x-0 bottom-[8%] will-change-[opacity,transform] lg:bottom-[10%]"
            style={chapterStyle(index)}
            aria-hidden={activeChapter !== index}
          >
            <div class:mx-auto={index === 0} class:text-center={index === 0} class="max-w-4xl">
              <p class="font-mono text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-primary">{chapter.eyebrow}</p>
              <h1 class={`mt-4 max-w-4xl font-semibold leading-[0.92] tracking-[-0.055em] ${index === 0 ? 'text-6xl sm:text-8xl' : 'text-5xl sm:text-7xl'}`}>
                {chapter.title}
              </h1>
              <p class:mx-auto={index === 0} class="mt-5 max-w-xl text-base leading-7 text-[var(--brand-text-secondary)] sm:text-lg">
                {chapter.body}
              </p>
              {#if index === 0}
                <p class="mt-7 font-mono text-[0.65rem] uppercase tracking-[0.2em] text-[var(--brand-text-tertiary)]">Scroll to operate ↓</p>
              {/if}
            </div>
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
