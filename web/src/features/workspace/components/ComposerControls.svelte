<script lang="ts">
  import type { Snippet } from 'svelte';
  import Icon from '../../../shared/ui/Icon.svelte';
  import type { Project } from '../logic/task-types';
  import { acceptedAttachmentTypes } from '../logic/attachment-drop';

  interface Props {
    id: string;
    projects?: Project[] | undefined;
    project?: string | undefined;
    attachable: boolean;
    disabled: boolean;
    busy: boolean;
    tools?: Snippet | undefined;
    onattach: (event: Event) => void;
  }

  let { id, projects, project = $bindable(''), attachable, disabled, busy, tools, onattach }: Props = $props();
</script>

{#if attachable}
  <label class="btn btn-ghost btn-circle btn-sm has-[:focus-visible]:outline has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-primary" title="Attach files">
    <Icon name="paperclip" class="size-4" />
    <span class="sr-only">Attach files</span>
    <input type="file" class="sr-only" accept={acceptedAttachmentTypes} multiple disabled={disabled || busy} onchange={onattach} />
  </label>
{/if}
{#if projects !== undefined}
  <label class="sr-only" for="{id}-project">Project</label>
  <select
    id="{id}-project"
    class="select select-ghost select-sm w-auto rounded-full {project === '' ? 'text-[var(--brand-text-secondary)]' : 'font-medium'}"
    bind:value={project}
    disabled={disabled || busy}
  >
    <option value="" disabled>Select a project</option>
    {#each projects as item (item.id)}<option value={item.id}>{item.name}</option>{/each}
  </select>
{/if}
{#if tools !== undefined}{@render tools()}{/if}
