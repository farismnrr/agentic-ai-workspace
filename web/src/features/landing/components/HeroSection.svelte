<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import Illustration from '../../../shared/ui/Illustration.svelte';
  import Marker from '../../../shared/ui/Marker.svelte';
  import { illustrations } from '../assets/illustrations';
  import { heroChecklist, reassurances, REPO_URL } from '../content';
  import TaskChecklist from './TaskChecklist.svelte';
</script>

<section class="mx-auto grid w-full max-w-6xl items-center gap-10 px-6 pb-16 pt-10 lg:grid-cols-[1fr_1.05fr] lg:gap-6 lg:pb-20 lg:pt-14">
  <div>
    <p class="text-[var(--brand-text-secondary)]">For people who aren’t developers</p>
    <h1 class="mt-5 font-display text-5xl font-medium leading-[1.05] tracking-tight sm:text-6xl">
      Tell it what you need.<br />
      See it done.<br />
      And <Marker>proven.</Marker>
    </h1>
    <p class="mt-7 max-w-md text-lg leading-relaxed text-[var(--brand-text-secondary)]">
      A small AI team does the work, shows every step, and checks that it really works before you accept it.
    </p>
    <div class="mt-8 flex flex-wrap items-center gap-x-8 gap-y-4">
      <a class="btn btn-primary btn-lg rounded-full" href={REPO_URL}>Get started, it’s free</a>
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

  <div class="relative mx-auto w-full max-w-xl lg:max-w-none">
    <Illustration asset={illustrations.deskLamp} class="h-auto w-full" priority />
    <TaskChecklist
      title="Fix the sign-in page on my website"
      items={heroChecklist}
      class="mt-6 w-full max-w-sm sm:absolute sm:bottom-[14%] sm:right-[17%] sm:mt-0 sm:w-[52%]"
    />
  </div>
</section>
