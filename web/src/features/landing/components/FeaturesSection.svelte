<script lang="ts">
  import { reveal } from '../../../shared/motion/reveal';
  import Illustration from '../../../shared/ui/Illustration.svelte';
  import Marker from '../../../shared/ui/Marker.svelte';
  import { illustrations } from '../assets/illustrations';
  import FinishedToast from './FinishedToast.svelte';
</script>

<section class="mx-auto w-full max-w-6xl px-6 py-14 lg:py-20">
  <p class="text-[var(--brand-text-secondary)]" use:reveal>Features</p>
  <div class="mt-3 grid gap-12 lg:grid-cols-[1.5fr_1fr] lg:gap-0">
    <div class="lg:pr-12" use:reveal>
      <h2 class="font-display text-4xl font-medium leading-tight tracking-tight sm:text-5xl">
        Made to fit <Marker>your day.</Marker>
      </h2>
      <div class="relative mt-8">
        <Illustration asset={illustrations.deskLaptop} class="h-auto w-full max-w-md" />
        <FinishedToast class="absolute right-0 top-[18%] w-[min(13rem,58%)] sm:right-[8%]" />
      </div>
      <h3 class="mt-6 text-3xl font-medium leading-tight tracking-tight sm:text-4xl">
        Step away.<br />It keeps going.
      </h3>
      <p class="mt-3 max-w-sm text-lg text-[var(--brand-text-secondary)]">
        Keep your computer on, grab lunch, and come back to finished work.
      </p>
    </div>

    <div class="flex flex-col divide-y divide-[var(--brand-border-strong)] border-[var(--brand-border-strong)] lg:border-l lg:pl-12">
      <div class="pb-10" use:reveal={120}>
        <Illustration asset={illustrations.key} class="h-auto w-48" />
        <h3 class="mt-5 text-3xl font-medium leading-tight tracking-tight">Your files stay home.</h3>
        <p class="mt-2 text-[var(--brand-text-secondary)]">It runs on your own computer, not someone else's server.</p>
      </div>
      <div class="flex items-end justify-between gap-4 pt-10" use:reveal={200}>
        <div>
          <h3 class="font-display text-3xl font-medium leading-tight tracking-tight">Free, and<br />always open.</h3>
          <p class="mt-2 text-[var(--brand-text-secondary)]">No subscription. Anyone can see how it works.</p>
        </div>
        <Illustration asset={illustrations.openSign} class="h-auto w-28 shrink-0" />
      </div>
    </div>
  </div>
</section>
