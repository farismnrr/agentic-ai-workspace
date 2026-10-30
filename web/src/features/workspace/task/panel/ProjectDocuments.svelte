<script lang="ts">
  import DocText from '../../components/DocText.svelte';
  import EmptyState from '../../../../shared/ui/EmptyState.svelte';
  import { rawUrl, readFile } from '../../logic/files-api';
  import { loadDocuments } from '../../logic/tasks-api';
  import type { ProjectDocument } from '../../logic/tasks-api';

  interface Props { taskId: string; projectId: string; updatedAt: string }
  const { taskId, projectId, updatedAt }: Props = $props();
  let documents = $state<ProjectDocument[]>([]);
  let error = $state('');
  let busy = $state(false);
  let selected = $state('');
  let content = $state('');
  let truncated = $state(false);
  let reading = $state(false);

  async function open(path: string): Promise<void> {
    selected = path;
    content = '';
    reading = true;
    error = '';
    try {
      const file = await readFile(projectId, path);
      if (selected !== path) return;
      content = file.content;
      truncated = file.truncated;
    } catch {
      if (selected === path) error = 'This document could not be opened. Try again.';
    } finally {
      if (selected === path) reading = false;
    }
  }

  $effect(() => {
    const current = { id: taskId, updatedAt };
    let active = true;
    busy = true;
    error = '';
    void loadDocuments(current.id).then((items) => {
      if (active) documents = items;
    }).catch(() => {
      if (active) error = 'Project documents could not be loaded. Reopen the Documents tab to retry.';
    }).finally(() => {
      if (active) busy = false;
    });
    return () => { active = false; };
  });
</script>

<section aria-label="Project documents">
  <h2 class="border-b border-[var(--brand-border)] pb-3 font-semibold">Project documents {#if busy}<span class="loading loading-spinner loading-xs"></span>{/if}</h2>
  <p class="my-2 text-sm text-base-content/70">Plans, research and long messages saved in this project. Every task can read them. Earlier plans may still be drafts.</p>
  {#if error !== ''}<p class="text-sm text-error" role="alert">{error}</p>{/if}
  {#if documents.length === 0 && !busy && error === ''}<EmptyState compact icon="file-text" title="No documents yet" body="Plans, research and long messages appear here when they're saved." />{/if}
  <ul class="flex flex-col divide-y divide-[var(--brand-border)] text-sm">
    {#each documents as document (document.path)}
      <li class="py-3">
        <button type="button" class="link link-primary text-left" onclick={() => void open(document.path)}>{document.path.split('/').slice(3).join('/')}</button>
        <p class="text-xs text-base-content/70">{document.title} · {document.status} · {new Date(document.updatedAt).toLocaleString()}</p>
      </li>
    {/each}
  </ul>
  {#if selected !== ''}
    <div class="mt-3 border-t border-base-300 pt-3">
      <div class="mb-2 flex items-center gap-2"><p class="min-w-0 flex-1 text-xs">{selected}</p><a class="link text-sm" href={rawUrl(projectId, selected)}>Download</a><button type="button" class="btn btn-ghost btn-xs" onclick={() => { selected = ''; }}>Close</button></div>
      {#if reading}<span class="loading loading-spinner loading-sm"></span>{:else}<DocText text={content} />{/if}
      {#if truncated}<p class="mt-2 text-sm text-base-content/70">Preview shortened. Download the complete file.</p>{/if}
    </div>
  {/if}
</section>
