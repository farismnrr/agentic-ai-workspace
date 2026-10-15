<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import Illustration from '../../../shared/ui/Illustration.svelte';
  import Marker from '../../../shared/ui/Marker.svelte';
  import { illustrations } from '../assets/illustrations';
  import { heroChecklist, reassurances } from '../content';
  import TaskChecklist from './TaskChecklist.svelte';
</script>

<section class="mx-auto grid grid-cols-1 w-full max-w-6xl items-center gap-10 px-6 pb-16 pt-10 lg:grid-cols-[1fr_1.05fr] lg:gap-6 lg:pb-20 lg:pt-14">
  <div class="hero-in">
    <p class="text-[var(--brand-text-secondary)]">For people who aren’t developers</p>
    <h1 class="mt-5 text-4xl font-medium leading-[1.05] tracking-tight min-[400px]:text-5xl sm:text-6xl">
      Tell it what you need.<br />
      See it done.<br />
      And <Marker>proven.</Marker>
    </h1>
    <p class="mt-7 max-w-md text-lg leading-relaxed text-[var(--brand-text-secondary)]">
      A small AI team does the work, shows every step, and checks that it really works before you accept it.
    </p>
    <div class="mt-8 flex flex-wrap items-center gap-x-8 gap-y-4">
      <a class="btn btn-primary btn-lg min-h-12 rounded-full" href="/register">Get started, it’s free</a>
      <a class="link link-hover inline-flex items-center gap-2" href="#how-it-works">
        See how it works <Icon name="arrow-right" class="size-4" />
      </a>
    </div>
    <ul class="mt-8 flex flex-wrap items-center gap-x-5 gap-y-2 text-sm text-[var(--brand-text-secondary)]">
      {#each reassurances as item (item.label)}
        <li class="flex items-center gap-2"><Icon name={item.icon} class="size-4" />{item.label}</li>
      {/each}
    </ul>
  </div>

  <div class="hero-in relative mx-auto w-full max-w-xl lg:max-w-none" style="--reveal-delay: 150ms">
    <Illustration asset={illustrations.deskLamp} class="h-auto w-full" priority />
    <TaskChecklist
      title="Fix the sign-in page on my website"
      items={heroChecklist}
      class="mt-6 w-full max-w-sm md:absolute md:bottom-[14%] md:right-[17%] md:mt-0 md:w-[52%]"
    />
  </div>
</section>
