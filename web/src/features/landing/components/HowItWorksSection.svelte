<script lang="ts">
  import { reveal } from '../../../shared/motion/reveal';
  import Icon from '../../../shared/ui/Icon.svelte';
  import Illustration from '../../../shared/ui/Illustration.svelte';
  import Marker from '../../../shared/ui/Marker.svelte';
  import { illustrations } from '../assets/illustrations';
  import { stepChecklist, steps } from '../content';
  import AcceptFormPreview from './AcceptFormPreview.svelte';
  import TaskChecklist from './TaskChecklist.svelte';
</script>

<section id="how-it-works" class="mx-auto w-full max-w-6xl px-6 py-14 lg:py-20">
  <div use:reveal>
  <p class="text-[var(--brand-text-secondary)]">How it works</p>
  <h2 class="mt-3 text-4xl font-medium leading-tight tracking-tight sm:text-5xl">
    Three steps. <Marker>No tech skills</Marker> needed.
  </h2>
  <p class="mt-4 text-lg text-[var(--brand-text-secondary)]">You talk, your AI team works, and you only accept what's proven.</p>
  </div>

  <ol class="mt-14 grid gap-14 md:grid-cols-2 lg:grid-cols-3 lg:gap-10">
    {#each steps as step, index (step.number)}
      <li class="relative" use:reveal={index * 120}>
        <div class="relative flex h-64 items-center justify-center sm:h-72">
          {#if index === 0}
            <Illustration asset={illustrations.smartphone} class="h-auto w-full max-w-xs" />
            <p class="card absolute right-0 top-1/4 max-w-[75%] flex-row items-start gap-3 border border-[var(--brand-border)] bg-base-100 p-3 text-sm shadow-[var(--brand-shadow)]">
              <Icon name="chat" class="mt-0.5 size-4 shrink-0" />
              <span>Make my contact form<br />send emails again</span>
            </p>
          {:else if index === 1}
            <Illustration asset={illustrations.papersPencil} class="mt-16 h-auto w-full max-w-xs" />
            <TaskChecklist items={stepChecklist} class="absolute left-0 top-0 w-[min(15rem,85%)] sm:left-4" />
          {:else}
            <Illustration asset={illustrations.rubberStamp} class="mr-auto h-4/5 w-auto" />
            <AcceptFormPreview class="absolute right-0 top-0 w-[min(9rem,42%)]" />
            <Illustration asset={illustrations.stampProven} class="absolute bottom-0 right-2 h-auto w-[min(11rem,52%)] -rotate-6" />
          {/if}
        </div>
        {#if index < steps.length - 1}
          <Icon name="arrow-right" class="absolute -right-8 top-1/3 hidden size-6 text-primary lg:block" />
        {/if}
        <p class="mt-8 font-display text-6xl font-medium text-primary" aria-hidden="true">{step.number}</p>
        <h3 class="mt-1 text-3xl font-medium leading-tight tracking-tight">{step.title}</h3>
        <p class="mt-2 text-[var(--brand-text-secondary)]">{step.text}</p>
      </li>
    {/each}
  </ol>
</section>
