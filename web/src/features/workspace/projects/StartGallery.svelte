<script lang="ts">
  import { templatePreview } from '../assets/template-previews';
  import { emptyStart, githubStart, newProjectCopy, templateStarts } from '../content/templates-copy';
  import Icon from '../../../shared/ui/Icon.svelte';

  interface Props {
    /** off: the server has no GitHub App; connect: it has one but the person has not connected; ready: connected. */
    github: 'off' | 'connect' | 'ready';
    onpick: (id: string) => void;
  }

  const { github, onpick }: Props = $props();
  const card = 'card card-border bg-base-100 text-left transition-colors hover:border-primary focus-visible:outline-2 focus-visible:outline-primary';
</script>

<h2 class="text-lg font-semibold">{newProjectCopy.startHeading}</h2>
<p class="mt-1 max-w-2xl text-[var(--brand-text-secondary)]">{newProjectCopy.startLead}</p>

<ul class="mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
  {#each templateStarts as start (start.id)}
    <li>
      <button type="button" class="{card} w-full overflow-hidden" onclick={() => { onpick(start.id); }}>
        <img class="aspect-[16/10] w-full border-b border-[var(--brand-border)] object-cover object-top" src={templatePreview(start.id)} alt="Screenshot of the {start.title} template" width="960" height="600" loading="lazy" />
        <span class="block p-4"><span class="block font-medium">{start.title}</span>
<span class="mt-1 block text-sm text-[var(--brand-text-secondary)]">{start.body}</span></span>
      </button>
    </li>
  {/each}
  <li>
    <button type="button" class="{card} h-full w-full p-4" onclick={() => { onpick(emptyStart.id); }}>
      <Icon name="plus" class="mb-3 size-6 text-primary" />
      <span class="block font-medium">{emptyStart.title}</span>
<span class="mt-1 block text-sm text-[var(--brand-text-secondary)]">{emptyStart.body}</span>
    </button>
  </li>
  {#if github !== 'off'}
    <li>
      {#if github === 'ready'}
        <button type="button" class="{card} h-full w-full p-4" onclick={() => { onpick(githubStart.id); }}>
          <Icon name="folder" class="mb-3 size-6 text-primary" />
          <span class="block font-medium">{githubStart.title}</span>
<span class="mt-1 block text-sm text-[var(--brand-text-secondary)]">{githubStart.body}</span>
        </button>
      {:else}
        <div class="card card-border h-full border-dashed bg-base-100 p-4">
          <Icon name="folder" class="mb-3 size-6 opacity-50" />
          <span class="block font-medium">{githubStart.title}</span>
<span class="mt-1 block text-sm text-[var(--brand-text-secondary)]">{githubStart.body}</span>
          <p class="mt-2 text-sm">{newProjectCopy.connectHint} <a class="link link-primary font-medium" href="/app/settings/integrations">{newProjectCopy.connectLink}</a></p>
        </div>
      {/if}
    </li>
  {/if}
</ul>
