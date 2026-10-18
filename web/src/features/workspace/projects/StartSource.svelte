<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import { githubStart, newProjectCopy, templateSource } from '../content/templates-copy';

  interface Props {
    /** connect: the server has a GitHub App but the person has not connected; ready: connected. */
    github: 'connect' | 'ready';
    onpick: (source: 'template' | 'github') => void;
  }

  const { github, onpick }: Props = $props();
  const card = 'card card-border bg-base-100 p-5 text-left';
  const live = `${card} transition-colors hover:border-primary focus-visible:outline-2 focus-visible:outline-primary`;
</script>

<h2 class="text-lg font-semibold">{newProjectCopy.sourceHeading}</h2>

<div class="mt-4 grid max-w-2xl gap-4 sm:grid-cols-2">
  <button type="button" class={live} onclick={() => { onpick('template'); }}>
    <Icon name="plus" class="mb-3 size-6 text-primary" />
    <span class="font-medium">{templateSource.title}</span>
    <span class="mt-1 text-sm text-[var(--brand-text-secondary)]">{templateSource.body}</span>
  </button>
  {#if github === 'ready'}
    <button type="button" class={live} onclick={() => { onpick('github'); }}>
      <Icon name="folder" class="mb-3 size-6 text-primary" />
      <span class="font-medium">{githubStart.title}</span>
      <span class="mt-1 text-sm text-[var(--brand-text-secondary)]">{githubStart.body}</span>
    </button>
  {:else}
    <div class="{card} border-dashed">
      <Icon name="folder" class="mb-3 size-6 opacity-50" />
      <span class="font-medium">{githubStart.title}</span>
      <span class="mt-1 text-sm text-[var(--brand-text-secondary)]">{githubStart.body}</span>
      <span class="mt-2 text-sm">{newProjectCopy.connectHint} <a class="link link-primary font-medium" href="/app/settings/integrations">{newProjectCopy.connectLink}</a></span>
    </div>
  {/if}
</div>
