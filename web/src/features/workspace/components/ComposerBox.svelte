<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import { projects } from '../content/projects';
  import { askLevels } from '../content/user';
  import Notice from './Notice.svelte';

  interface Props {
    id: string;
    placeholder: string;
    text?: string | undefined;
    withProject?: boolean;
  }

  let { id, placeholder, text = $bindable(''), withProject = false }: Props = $props();
  let project = $state('bakery');
  let ask = $state('always');
  let sent = $state(false);

  function onsubmit(event: SubmitEvent): void {
    event.preventDefault();
    sent = text.trim() !== '';
  }
</script>

<form {onsubmit}>
  <label class="sr-only" for={id}>{placeholder}</label>
  <div class="rounded-3xl border border-[var(--brand-border)] bg-base-100 shadow-lg focus-within:border-[var(--brand-border-strong)]">
    <textarea {id} bind:value={text} rows="2" {placeholder} class="field-sizing-content max-h-52 min-h-14 w-full resize-none bg-transparent px-4 pb-1 pt-3.5 text-[15px] focus:outline-none"></textarea>
    <div class="flex flex-wrap items-center gap-1 px-2 pb-2">
      <button class="btn btn-ghost btn-circle btn-sm" type="button" aria-label="Attach a file" title="Attach a file (preview)"><Icon name="paperclip" class="size-4" /></button>
      {#if withProject}
        <label class="sr-only" for="{id}-project">Project</label>
        <select id="{id}-project" class="select select-ghost select-sm w-auto rounded-full" bind:value={project}>
          {#each projects as item (item.id)}<option value={item.id}>{item.name}</option>{/each}
        </select>
      {/if}
      <label class="sr-only" for="{id}-ask">How much should it ask you?</label>
      <select id="{id}-ask" class="select select-ghost select-sm ml-auto w-auto rounded-full" bind:value={ask}>
        {#each askLevels as level (level.value)}<option value={level.value}>{level.label}</option>{/each}
      </select>
      <button class="btn btn-primary btn-circle btn-sm" type="submit" aria-label="Send"><Icon name="arrow-right" class="size-4" /></button>
    </div>
  </div>
  {#if sent}<div class="mt-2"><Notice show /></div>{/if}
</form>
