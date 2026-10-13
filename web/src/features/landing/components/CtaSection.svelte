<script lang="ts">
  import { reveal } from '../../../shared/motion/reveal';
  import Illustration from '../../../shared/ui/Illustration.svelte';
  import Icon from '../../../shared/ui/Icon.svelte';
  import Marker from '../../../shared/ui/Marker.svelte';
  import { illustrations } from '../assets/illustrations';
  import { reassurances } from '../content';
</script>

<section id="get-started" class="mx-auto w-full max-w-6xl px-6 pb-16">
  <div use:reveal class="card overflow-hidden rounded-[2rem] border border-[var(--brand-border-strong)] bg-base-300 lg:card-side">
    <div class="card-body justify-center gap-0 p-8 sm:p-12">
      <h2 class="font-display text-3xl font-medium leading-tight tracking-tight min-[400px]:text-4xl sm:text-5xl">
        Ready when <Marker>you are.</Marker>
      </h2>
      <p class="mt-3 text-xl text-[var(--brand-text-secondary)]">Tell it what you need. It’ll take it from there.</p>
      <div class="mt-7">
        <a class="btn btn-primary btn-lg min-h-12 rounded-full" href="/register">Get started, it’s free</a>
      </div>
      <ul class="mt-6 flex flex-wrap items-center gap-x-5 gap-y-2 text-sm text-[var(--brand-text-secondary)]">
        {#each reassurances as item (item.label)}
          <li class="flex items-center gap-2"><Icon name={item.icon} class="size-4" />{item.label}</li>
        {/each}
      </ul>
    </div>
    <figure class="flex items-end justify-center px-8 pt-2 lg:justify-end lg:pr-12">
      <Illustration asset={illustrations.openDoor} class="h-auto w-48 sm:w-56" />
    </figure>
  </div>
</section>
