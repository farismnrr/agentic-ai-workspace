<script lang="ts">
  import { newProjectCopy } from '../content/templates-copy';
  import Icon from '../../../shared/ui/Icon.svelte';

  interface Props {
    /** off: no GitHub App on the server, so a new repository is not offered; connect: not connected yet. */
    github: 'off' | 'connect' | 'ready';
    onpick: (source: 'empty' | 'new') => void;
  }

  const { github, onpick }: Props = $props();
  const card = 'card card-border bg-base-100 p-5 text-left';
</script>

<h2 class="text-lg font-semibold">{newProjectCopy.whereHeading}</h2>

<div class="mt-4 grid max-w-2xl gap-4 sm:grid-cols-2">
  <button type="button" class="{card} transition-colors hover:border-primary focus-visible:outline-2 focus-visible:outline-primary" onclick={() => { onpick('empty'); }}>
    <Icon name="folder" class="mb-3 size-6 text-primary" />
    <span class="font-medium">{newProjectCopy.project.title}</span>
    <span class="mt-1 text-sm text-[var(--brand-text-secondary)]">{newProjectCopy.project.body}</span>
  </button>
  {#if github === 'ready'}
    <button type="button" class="{card} transition-colors hover:border-primary focus-visible:outline-2 focus-visible:outline-primary" onclick={() => { onpick('new'); }}>
      <Icon name="plus" class="mb-3 size-6 text-primary" />
      <span class="font-medium">{newProjectCopy.repo.title}</span>
      <span class="mt-1 text-sm text-[var(--brand-text-secondary)]">{newProjectCopy.repo.body}</span>
    </button>
  {:else if github === 'connect'}
    <div class="{card} border-dashed">
      <Icon name="plus" class="mb-3 size-6 opacity-50" />
      <span class="font-medium">{newProjectCopy.repo.title}</span>
      <span class="mt-1 text-sm text-[var(--brand-text-secondary)]">{newProjectCopy.repo.body}</span>
      <span class="mt-2 text-sm">{newProjectCopy.connectHint} <a class="link link-primary font-medium" href="/app/settings/integrations">{newProjectCopy.connectLink}</a></span>
    </div>
  {/if}
</div>
