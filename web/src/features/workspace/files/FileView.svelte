<script lang="ts">
  import { untrack } from 'svelte';
  import Icon from '../../../shared/ui/Icon.svelte';
  import EmptyState from '../../../shared/ui/EmptyState.svelte';
  import CodeEditor from '../components/CodeEditor.svelte';
  import BinaryFile from './BinaryFile.svelte';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import type { FileView } from '../logic/file-view';

  interface Props {
    view: FileView;
    projectId: string;
    gitVersion: string;
    /** Back to the tree, on a screen too narrow to show both. */
    onback: () => void;
    ondeleted: () => void;
  }

  const { view, projectId, gitVersion, onback, ondeleted }: Props = $props();
  // A view never changes for the life of this component: the browser makes a new one per project.
  const shown = untrack(() => view.state);
  const binaryPreview = $derived(
    $shown.file?.binary === true ||
      ['pdf', 'png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'mp4', 'webm', 'mov']
        .includes($shown.path?.split('.').pop()?.toLowerCase() ?? ''),
  );
  const dirty = $derived($shown.file !== undefined && $shown.draft !== $shown.file.content);
  const editable = $derived($shown.file !== undefined && !binaryPreview && !$shown.file.truncated);
  const deleteWarning = $derived(
    binaryPreview || $shown.file?.truncated === true
      ? 'This file is removed permanently and cannot be restored with Undo. Keep a copy if unsure.'
      : 'It is removed from the project. The AI or you can bring it back with Undo while its change is listed.',
  );
  let confirming = $state(false);

  $effect(() => {
    const version = gitVersion;
    if (version !== '') untrack(() => void view.refresh());
  });

  // On the window, not the section: pressing a button in it (Keep mine) can remove that button and with it the focus.
  function onkeydown(event: KeyboardEvent): void {
    if ($shown.path !== undefined && (event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 's') {
      event.preventDefault();
      if (dirty) void view.save();
    }
  }

  async function remove(): Promise<void> {
    confirming = false;
    await view.remove();
    ondeleted();
  }
</script>

<svelte:window {onkeydown} />

<section class="flex h-full min-h-0 min-w-0 flex-col" aria-label="File">
  {#if $shown.path !== undefined}
    <header class="flex flex-wrap items-center gap-2 border-b border-[var(--brand-border)] px-3 py-2">
      <button type="button" class="btn btn-ghost btn-xs @2xl:hidden" onclick={onback} aria-label="Back to files"><Icon name="arrow-left" class="size-4" /></button>
      <code class="min-w-0 flex-1 truncate text-xs">{$shown.path}</code>
      {#if dirty}<span class="badge badge-warning badge-sm">Unsaved</span>{:else if $shown.file !== undefined}<span class="badge badge-ghost badge-sm">Saved</span>{/if}
      {#if editable}<button type="button" class="btn btn-primary btn-xs" disabled={!dirty || $shown.saving} onclick={() => void view.save()}>{$shown.saving ? 'Saving…' : 'Save'}</button>{/if}
      <button type="button" class="btn btn-ghost btn-xs text-error" aria-label="Delete file" onclick={() => { confirming = true; }}><Icon name="trash" class="size-4" /></button>
    </header>
    {#if $shown.diverged}
      <div class="alert alert-warning alert-soft m-2 text-sm" role="alert">
        <div class="flex min-w-0 flex-col gap-2">
          <span>This file changed on disk while you were editing it.</span>
          <div class="flex flex-wrap gap-2">
            <button type="button" class="btn btn-xs whitespace-nowrap" onclick={view.loadTheirs}>Show the new version</button>
            <button type="button" class="btn btn-xs btn-primary whitespace-nowrap" onclick={view.keepMine}>Keep mine</button>
          </div>
        </div>
      </div>
    {/if}
    {#if $shown.error !== ''}<div class="alert alert-error alert-soft m-2 text-sm" role="alert">{$shown.error}</div>{/if}
    {#if $shown.loading}
      <div class="grid flex-1 place-items-center"><span class="loading loading-spinner"></span></div>
    {:else if binaryPreview}
      <BinaryFile {projectId} path={$shown.path} />
    {:else if $shown.file !== undefined}
      {#if $shown.file.truncated}<p class="mx-3 mt-2 text-xs text-[var(--brand-text-secondary)]">This file is large, so only the start is shown and it cannot be edited here.</p>{/if}
      <div class="min-h-0 flex-1">
        {#key $shown.path}<CodeEditor path={$shown.path} value={$shown.draft} original={$shown.head?.available && !$shown.head.binary && !$shown.head.truncated ? $shown.head.content : undefined} readOnly={!editable} onchange={view.edit} />{/key}
      </div>
    {/if}
  {:else}
    <EmptyState compact icon="file-text" title="Choose a file" body="Pick a file to read or edit it. Changes by the AI appear here as they happen." />
  {/if}
</section>

<ConfirmModal open={confirming} title={`Delete ${$shown.path?.split('/').pop() ?? 'this file'}?`} body={deleteWarning} confirmLabel="Delete" cancelLabel="Keep" oncancel={() => { confirming = false; }} onconfirm={() => void remove()} />
