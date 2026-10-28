<script lang="ts">
  import { untrack } from 'svelte';
  import { ApiError } from '../../../shared/api/http';
  import { readFile, readGitHead, type GitFileStatus } from '../logic/files-api';
  import DiffView from './DiffView.svelte';

  interface Props { projectId: string; file: GitFileStatus; refreshKey: number }
  const { projectId, file, refreshKey }: Props = $props();
  let expanded = $state(false);
  let loading = $state(false);
  let problem = $state('');
  let before = $state<string>();
  let after = $state<string>();
  let loadRevision = 0;

  async function load(): Promise<void> {
    const requestId = ++loadRevision;
    loading = true;
    problem = '';
    before = undefined;
    after = undefined;
    const result = await readDiff(projectId, file);
    if (requestId !== loadRevision) return;
    before = result.before;
    after = result.after;
    problem = result.problem;
    loading = false;
  }

  async function readDiff(project: string, target: GitFileStatus): Promise<{ before?: string | undefined; after?: string | undefined; problem: string }> {
    try {
      const head = await readGitHead(project, target.originalPath ?? target.path);
      if (!head.available) return { problem: 'Git comparison is not available for this file.' };
      if (head.binary || head.truncated) return { problem: 'This file is binary or too large to compare as text.' };
      const current = await readCurrent(project, target);
      return current.problem === ''
        ? { before: head.content, after: current.content, problem: '' }
        : { problem: current.problem };
    } catch {
      return { problem: 'The Git comparison could not be loaded.' };
    }
  }

  async function readCurrent(project: string, target: GitFileStatus): Promise<{ content?: string | undefined; problem: string }> {
    try {
      const current = await readFile(project, target.path);
      if (current.binary || current.truncated) return { problem: 'This file is binary or too large to compare as text.' };
      return { content: current.content, problem: '' };
    } catch (error) {
      if (error instanceof ApiError && error.code === 'not_found' && `${target.index}${target.worktree}`.includes('D')) {
        return { content: '', problem: '' };
      }
      return { problem: 'The current file could not be read.' };
    }
  }

  function toggle(): void {
    expanded = !expanded;
    if (expanded) void load();
  }

  $effect(() => {
    const version = refreshKey;
    if (version > 0 && expanded) untrack(() => void load());
  });
</script>

<div class="min-w-0">
  <button class="btn btn-ghost btn-xs" type="button" aria-expanded={expanded} onclick={toggle}>
    {expanded ? 'Hide diff' : 'Show diff'}
  </button>
  {#if expanded}
    {#if loading}<span class="loading loading-spinner loading-xs ml-2" aria-label="Loading Git diff"></span>
    {:else if problem !== ''}<p class="mt-2 text-xs text-[var(--brand-text-secondary)]" role="status">{problem}</p>
    {:else if before !== undefined && after !== undefined}
      <div class="mt-2 max-h-80 overflow-auto rounded border border-[var(--brand-border)]"><DiffView path={file.path} {before} {after} /></div>
    {/if}
  {/if}
</div>
