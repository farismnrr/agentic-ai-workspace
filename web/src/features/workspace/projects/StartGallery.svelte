<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import { templatePreview } from '../assets/template-previews';
  import { emptyStart, newProjectCopy, templateStarts } from '../content/templates-copy';

  interface Props {
    onpick: (id: string) => void;
  }

  const { onpick }: Props = $props();
  const card = 'card card-border bg-base-100 text-left transition-colors hover:border-primary focus-visible:outline-2 focus-visible:outline-primary';
</script>

<h2 class="text-lg font-semibold">{newProjectCopy.startHeading}</h2>
<p class="mt-1 max-w-2xl text-[var(--brand-text-secondary)]">{newProjectCopy.startLead}</p>

<ul class="mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
  <li>
    <button type="button" class="{card} h-full w-full p-4" onclick={() => { onpick(emptyStart.id); }}>
      <Icon name="plus" class="mb-3 size-6 text-primary" />
      <span class="block font-medium">{emptyStart.title}</span>
      <span class="mt-1 block text-sm text-[var(--brand-text-secondary)]">{emptyStart.body}</span>
    </button>
  </li>
  {#each templateStarts as start (start.id)}
    <li>
      <button type="button" class="{card} w-full overflow-hidden" onclick={() => { onpick(start.id); }}>
        <img class="aspect-[16/10] w-full border-b border-[var(--brand-border)] object-cover object-top" src={templatePreview(start.id)} alt="Screenshot of the {start.title} template" width="960" height="600" loading="lazy" />
        <span class="block p-4">
          <span class="block font-medium">{start.title}</span>
          <span class="mt-1 block text-sm text-[var(--brand-text-secondary)]">{start.body}</span>
        </span>
      </button>
    </li>
  {/each}
</ul>
