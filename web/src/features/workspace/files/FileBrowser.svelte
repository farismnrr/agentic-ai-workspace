<script lang="ts">
  import { onMount, untrack } from 'svelte';
  import { ApiError } from '../../../shared/api/http';
  import Icon from '../../../shared/ui/Icon.svelte';
  import EmptyState from '../../../shared/ui/EmptyState.svelte';
  import { createFileStore } from '../logic/file-store';
  import { createFileView } from '../logic/file-view';
  import { deleteFolder, followFiles, saveFile } from '../logic/files-api';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import FileView from './FileView.svelte';
  import TreeLevel from './TreeLevel.svelte';

  interface Props {
    projectId: string;
    /** Paths the open task changed; they are marked in the tree. */
    changed?: ReadonlySet<string>;
    /** The height of the browser, as a class. */
    height?: string;
  }

  const { projectId, changed = new Set<string>(), height = 'h-[70vh]' }: Props = $props();
  const store = untrack(() => createFileStore(projectId));
  const view = untrack(() => createFileView(projectId));
  const { dirs, expanded } = store;
  const shown = view.state;
  let naming = $state(false);
  let newPath = $state('');
  let problem = $state('');
  let deleting = $state<string | undefined>();
  let deleteProblem = $state('');

  onMount(() => {
    let timer: ReturnType<typeof setTimeout> | undefined;
    const stop = followFiles(projectId, () => {
      // Many files change in a burst while the agent works: read once when it settles.
      clearTimeout(timer);
      timer = setTimeout(() => { void store.reload(); void view.refresh(); }, 300);
    });
    return () => { clearTimeout(timer); stop(); };
  });

  async function removeFolder(): Promise<void> {
    const folder = deleting;
    deleting = undefined;
    if (folder === undefined) return;
    try {
      await deleteFolder(projectId, folder);
      deleteProblem = '';
      if ($shown.path?.startsWith(`${folder}/`) === true) view.close();
      await store.reload();
    } catch {
      deleteProblem = 'That folder could not be deleted.';
    }
  }

  async function create(event: SubmitEvent): Promise<void> {
    event.preventDefault();
    const path = newPath.trim().replace(/^\/+/, '');
    if (path === '') return;
    try {
      await saveFile(projectId, path, '', '');
      naming = false;
      newPath = '';
      problem = '';
      await store.reload();
      await view.open(path);
    } catch (error) {
      problem = error instanceof ApiError && error.code === 'exists' ? 'A file with that name already exists.' : 'That name cannot be used.';
    }
  }
</script>

<div class="@container overflow-hidden {height}">
  <div class="grid h-full min-h-0 grid-cols-[minmax(0,1fr)] @2xl:grid-cols-[16rem_minmax(0,1fr)]">
    <nav class="min-h-0 flex-col overflow-y-auto border-[var(--brand-border)] p-2 @2xl:border-r {$shown.path === undefined ? 'flex' : 'hidden @2xl:flex'}" aria-label="Files">
      <div class="mb-1 flex items-center justify-between px-1">
        <span class="text-xs font-medium text-[var(--brand-text-secondary)]">Files</span>
        <button type="button" class="btn btn-ghost btn-xs" onclick={() => { naming = !naming; }}><Icon name="plus" class="size-3.5" /> New file</button>
      </div>
      {#if naming}
        <form class="mb-2 flex flex-col gap-1 px-1" onsubmit={create}>
          <input class="input input-sm w-full" placeholder="folder/name.txt" aria-label="Path of the new file" bind:value={newPath} />
          {#if problem !== ''}<p class="text-xs text-error" role="alert">{problem}</p>{/if}
        </form>
      {/if}
      {#if deleteProblem !== ''}<p class="px-1 pb-1 text-xs text-error" role="alert">{deleteProblem}</p>{/if}
      {#if $dirs['.'] === undefined}
        <div class="grid flex-1 place-items-center"><span class="loading loading-spinner loading-sm" aria-label="Loading files"></span></div>
      {:else if $dirs['.'].length === 0}
        <EmptyState compact icon="folder" title="No files yet" body="Create a file to get started." />
      {:else}
        <TreeLevel dirs={$dirs} expanded={$expanded} path="." depth={0} selected={$shown.path} {changed} ontoggle={(path: string) => void store.toggle(path)} onopen={(path: string) => void view.open(path)} ondeletefolder={(path: string) => { deleting = path; }} />
      {/if}
    </nav>
    <div class="min-h-0 min-w-0 {$shown.path === undefined ? 'hidden @2xl:block' : 'block'}">
      <FileView {view} {projectId} onback={view.close} ondeleted={() => void store.reload()} />
    </div>
  </div>
</div>

<ConfirmModal
  open={deleting !== undefined}
  title="Delete this folder?"
  body="The folder and everything inside it are removed from the project. This cannot be undone, so keep a copy if unsure."
  confirmLabel="Delete folder"
  cancelLabel="Keep"
  oncancel={() => { deleting = undefined; }}
  onconfirm={() => void removeFolder()}
/>
