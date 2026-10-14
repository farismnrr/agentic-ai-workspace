<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import { askLevels } from '../content/tasks-copy';
  import type { ApprovalMode, Project } from '../logic/task-types';

  interface Props {
    id: string;
    placeholder: string;
    text?: string | undefined;
    /** When given, the composer shows a project picker. */
    projects?: Project[] | undefined;
    project?: string | undefined;
    busy?: boolean | undefined;
    disabled?: boolean | undefined;
    /** Hide the "how much should it ask" choice, for follow-ups that keep the task's setting. */
    withMode?: boolean | undefined;
    error?: string | undefined;
    onsend: (text: string, mode: ApprovalMode, project: string) => void;
  }

  let { id, placeholder, text = $bindable(''), projects, project = $bindable(''), busy = false, disabled = false, withMode = true, error = '', onsend }: Props = $props();
  let mode: ApprovalMode = $state('risky');

  function onsubmit(event: SubmitEvent): void {
    event.preventDefault();
    if (text.trim() !== '' && !busy && !disabled) onsend(text.trim(), mode, project);
  }
</script>

<form {onsubmit}>
  <label class="sr-only" for={id}>{placeholder}</label>
  <div class="rounded-3xl border border-[var(--brand-border)] bg-base-100 shadow-lg focus-within:border-[var(--brand-border-strong)]">
    <textarea {id} bind:value={text} rows="2" {placeholder} {disabled} class="field-sizing-content max-h-52 min-h-14 w-full resize-none bg-transparent px-4 pb-1 pt-3.5 text-[15px] focus:outline-none"></textarea>
    <div class="flex flex-wrap items-center gap-1 px-2 pb-2">
      {#if projects !== undefined}
        <label class="sr-only" for="{id}-project">Project</label>
        <select id="{id}-project" class="select select-ghost select-sm w-auto rounded-full" bind:value={project} {disabled}>
          {#each projects as item (item.id)}<option value={item.id}>{item.name}</option>{/each}
        </select>
      {/if}
      {#if withMode}
        <label class="sr-only" for="{id}-ask">How much should it ask you?</label>
        <select id="{id}-ask" class="select select-ghost select-sm ml-auto w-auto rounded-full" bind:value={mode} {disabled}>
          {#each askLevels as level (level.value)}<option value={level.value}>{level.label}</option>{/each}
        </select>
      {/if}
      <button class="btn btn-primary btn-circle btn-sm {withMode ? '' : 'ml-auto'}" type="submit" aria-label="Send" disabled={busy || disabled || text.trim() === ''}>
        {#if busy}<span class="loading loading-spinner loading-xs"></span>{:else}<Icon name="arrow-right" class="size-4" />{/if}
      </button>
    </div>
  </div>
  {#if error !== ''}<p class="mt-2 text-sm text-error" role="alert">{error}</p>{/if}
</form>
