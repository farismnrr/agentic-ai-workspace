<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import { extensionsCopy } from '../content/extensions-copy';
  import type { Extension } from '../logic/extensions-api';

  interface Props {
    extension: Extension;
    installing: boolean;
    onremove: (extension: Extension) => void;
  }

  const { extension, installing, onremove }: Props = $props();
  const shown = 5;
  const abilities = $derived(extension.capabilities.slice(0, shown));
  const rest = $derived(Math.max(0, extension.capabilities.length - shown));
</script>

<li class="flex flex-col gap-3 px-4 py-4">
  <div class="flex flex-wrap items-start gap-3">
    <Icon name="puzzle" class="mt-1 size-5 shrink-0 text-primary" />
    <div class="min-w-0 flex-1">
      <p class="flex flex-wrap items-center gap-2 font-medium">
        <span class="truncate">{extension.name}</span>
        <span class="badge badge-sm badge-ghost">v{extension.version}</span>
        <span class="text-xs font-normal text-[var(--brand-text-tertiary)]">{extensionsCopy.size(extension.size)}</span>
      </p>
      {#if extension.description !== ''}<p class="mt-0.5 text-sm text-[var(--brand-text-secondary)]">{extension.description}</p>{/if}
    </div>
    <div class="flex items-center gap-2">
      <a class="btn btn-sm rounded-lg {installing ? 'btn-outline' : 'btn-primary'}" href="/app/extensions/{extension.projectId}">{extensionsCopy.install}</a>
      <a class="btn btn-outline btn-sm rounded-lg" href={extension.downloadUrl} download={extension.fileName}><Icon name="download" class="size-4" /> {extensionsCopy.download}</a>
      <button class="btn btn-ghost btn-sm text-error" type="button" aria-label="{extensionsCopy.remove} {extension.name}" onclick={() => { onremove(extension); }}><Icon name="trash" class="size-4" /></button>
    </div>
  </div>
  {#if abilities.length > 0}
    <div class="pl-8">
      <p class="text-xs font-medium text-[var(--brand-text-secondary)]">{extensionsCopy.canDo}</p>
      <ul class="mt-1 flex flex-col gap-1 text-sm">
        {#each abilities as ability (ability.text)}
          <li class="flex items-start gap-2">
            {#if ability.risk === 'high'}<span class="badge badge-warning badge-soft badge-sm shrink-0">Needs trust</span>{:else}<span class="badge badge-ghost badge-sm shrink-0">Light</span>{/if}
            <span>{ability.text}</span>
          </li>
        {/each}
        {#if rest > 0}<li class="text-xs text-[var(--brand-text-tertiary)]">{extensionsCopy.more(rest)}</li>{/if}
      </ul>
    </div>
  {/if}
</li>
