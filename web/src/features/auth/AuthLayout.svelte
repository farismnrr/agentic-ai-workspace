<script lang="ts">
  import type { Snippet } from 'svelte';
  import Illustration from '../../shared/ui/Illustration.svelte';
  import LogoMark from '../../shared/ui/LogoMark.svelte';
  import type { authIllustrations } from './assets/illustrations';

  interface Props {
    illustration: (typeof authIllustrations)[keyof typeof authIllustrations];
    caption: string;
    children: Snippet;
  }

  const { illustration, caption, children }: Props = $props();
</script>

<div class="grid min-h-screen bg-base-100 lg:grid-cols-2">
  <main class="flex flex-col px-6 py-8 sm:px-12 lg:px-[clamp(3rem,8vw,7.5rem)]">
    <a class="flex min-h-11 w-fit items-center gap-2.5" href="/" aria-label="Masih Awam home">
      <LogoMark class="size-10" />
      <span class="whitespace-nowrap text-[17px] font-semibold tracking-[-0.01em]">Masih Awam</span>
    </a>
    <div class="hero-in my-auto w-full max-w-md py-10">
      {@render children()}
    </div>
  </main>

  <aside class="hidden flex-col items-center justify-center gap-6 bg-[var(--brand-panel)] p-12 lg:flex">
    <Illustration asset={illustration} class="hero-in h-auto max-h-[52vh] w-auto max-w-full" priority />
    <p class="max-w-xs text-balance text-center font-display text-3xl font-medium leading-tight">{caption}</p>
  </aside>
</div>
