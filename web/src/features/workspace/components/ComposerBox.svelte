<script lang="ts">
  import type { Snippet } from 'svelte';
  import Icon from '../../../shared/ui/Icon.svelte';
  import type { Project } from '../logic/task-types';

  interface Props {
    id: string;
    placeholder: string;
    text?: string | undefined;
    /** When given, the composer asks which project the message is for. */
    projects?: Project[] | undefined;
    project?: string | undefined;
    busy?: boolean | undefined;
    disabled?: boolean | undefined;
    error?: string | undefined;
    /** When true, the person can attach pictures; they are kept here until the message is sent. */
    files?: File[] | undefined;
    attachable?: boolean | undefined;
    /** Extra controls in the toolbar, before the send button. */
    tools?: Snippet | undefined;
    onsend: (text: string, project: string) => void;
  }

  let {
    id,
    placeholder,
    text = $bindable(''),
    projects,
    project = $bindable(''),
    busy = false,
    disabled = false,
    error = '',
    files = $bindable([]),
    attachable = false,
    tools,
    onsend,
  }: Props = $props();

  const needsProject = $derived(projects !== undefined && project === '');
  const canSend = $derived(text.trim() !== '' && !needsProject && !busy && !disabled);

  function attach(event: Event): void {
    const input = event.currentTarget as HTMLInputElement;
    files = [...files, ...Array.from(input.files ?? [])];
    input.value = '';
  }

  function send(): void {
    if (canSend) onsend(text.trim(), project);
  }

  function onsubmit(event: SubmitEvent): void {
    event.preventDefault();
    send();
  }

  // Enter sends, Shift+Enter starts a new line; Enter while composing text (IME) is left alone.
  function onkeydown(event: KeyboardEvent): void {
    if (event.key !== 'Enter' || event.shiftKey || event.isComposing) return;
    event.preventDefault();
    send();
  }
</script>

<form {onsubmit}>
  <label class="sr-only" for={id}>{placeholder}</label>
  <div class="rounded-3xl border border-[var(--brand-border)] bg-base-100 shadow-lg focus-within:border-[var(--brand-border-strong)]">
    <textarea {id} bind:value={text} {onkeydown} rows="2" {placeholder} {disabled} class="field-sizing-content max-h-52 min-h-14 w-full resize-none bg-transparent px-4 pb-1 pt-3.5 text-[15px] focus:outline-none"></textarea>
    {#if files.length > 0}
      <ul class="flex flex-wrap gap-1 px-3 pb-1" aria-label="Attached pictures">
        {#each files as file, index (index)}
          <li class="badge badge-neutral badge-soft gap-1 py-3">
            <Icon name="paperclip" class="size-3" />{file.name}
            <button type="button" class="btn btn-ghost btn-xs btn-circle size-4 min-h-0 p-0" aria-label="Remove {file.name}" onclick={() => (files = files.filter((_, at) => at !== index))}><Icon name="x" class="size-3" /></button>
          </li>
        {/each}
      </ul>
    {/if}
    <div class="flex flex-wrap items-center gap-1 px-2 pb-2">
      {#if attachable}
        <label class="btn btn-ghost btn-circle btn-sm" title="Attach pictures">
          <Icon name="paperclip" class="size-4" /><span class="sr-only">Attach pictures</span>
          <input type="file" class="hidden" accept="image/png,image/jpeg,image/webp,image/gif,image/svg+xml" multiple {disabled} onchange={attach} />
        </label>
      {/if}
      {#if projects !== undefined}
        <label class="sr-only" for="{id}-project">Project</label>
        <select id="{id}-project" class="select select-ghost select-sm w-auto rounded-full {needsProject ? 'text-[var(--brand-text-secondary)]' : 'font-medium'}" bind:value={project} {disabled}>
          <option value="" disabled>Select a project</option>
          {#each projects as item (item.id)}<option value={item.id}>{item.name}</option>{/each}
        </select>
      {/if}
      {#if tools !== undefined}{@render tools()}{/if}
      <button class="btn btn-primary btn-circle btn-sm ml-auto" type="submit" aria-label="Send" disabled={!canSend}>
        {#if busy}<span class="loading loading-spinner loading-xs"></span>{:else}<Icon name="arrow-right" class="size-4" />{/if}
      </button>
    </div>
  </div>
  {#if needsProject && text.trim() !== ''}<p class="mt-2 text-sm text-[var(--brand-text-secondary)]">Select a project so I know where to work.</p>{/if}
  {#if error !== ''}<p class="mt-2 text-sm text-error" role="alert">{error}</p>{/if}
</form>
