<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import type { FileEntry } from '../logic/files-api';
  import { gitStatusByPath, gitStatusLabel } from '../logic/git-status-store';
  import type { GitFileStatus } from '../logic/files-api';
  import TreeLevel from './TreeLevel.svelte';

  interface Props {
    /** The entries of every folder read so far, by folder path. */
    dirs: Record<string, FileEntry[]>;
    expanded: ReadonlySet<string>;
    path: string;
    depth: number;
    selected: string | undefined;
    statuses: GitFileStatus[];
    ontoggle: (path: string) => void;
    onopen: (path: string) => void;
    ondeletefolder: (path: string) => void;
  }

  const { dirs, expanded, path, depth, selected, statuses, ontoggle, onopen, ondeletefolder }: Props = $props();
  const entries = $derived(dirs[path] ?? []);
  const statusByPath = $derived(gitStatusByPath(statuses));
  const touches = (folder: string): boolean => statuses.some((file) => file.path.startsWith(`${folder}/`) || (file.originalPath?.startsWith(`${folder}/`) ?? false));
  const code = (file: GitFileStatus): string => file.conflict ? '!' : file.untracked ? 'U' : file.index.includes('D') || file.worktree.includes('D') ? 'D' : file.index.includes('A') || file.worktree.includes('A') ? 'A' : file.originalPath || file.index.includes('R') || file.worktree.includes('R') ? 'R' : 'M';
</script>

<ul class="flex flex-col" role={depth === 0 ? 'tree' : 'group'} aria-label={depth === 0 ? 'Project files' : undefined}>
  {#each entries as entry (entry.path)}
    <li role="treeitem" aria-selected={selected === entry.path} aria-expanded={entry.dir ? expanded.has(entry.path) : undefined}>
      {#if entry.dir}
        <div class="group flex items-center rounded hover:bg-base-200">
          <button type="button" class="flex min-w-0 flex-1 items-center gap-1.5 rounded px-2 py-1 text-left text-sm" style:padding-left="{depth * 0.75 + 0.5}rem" onclick={() => { ontoggle(entry.path); }}>
            <Icon name={expanded.has(entry.path) ? 'chevron-down' : 'chevron-right'} class="size-3.5 shrink-0 text-[var(--brand-text-tertiary)]" />
            <Icon name="folder" class="size-4 shrink-0 text-[var(--brand-text-secondary)]" />
            <span class="min-w-0 flex-1 truncate">{entry.name}</span>
            {#if touches(entry.path)}<span class="badge badge-warning badge-xs" aria-label="Contains Git changes">•</span>{/if}
          </button>
          <button type="button" class="btn btn-ghost btn-xs mr-1 text-error opacity-0 focus-visible:opacity-100 group-hover:opacity-100" aria-label="Delete folder {entry.name}" onclick={() => { ondeletefolder(entry.path); }}><Icon name="trash" class="size-3.5" /></button>
        </div>
        {#if expanded.has(entry.path)}
          <TreeLevel {dirs} {expanded} path={entry.path} depth={depth + 1} {selected} {statuses} {ontoggle} {onopen} {ondeletefolder} />
        {/if}
      {:else}
        <button type="button" class="flex w-full items-center gap-1.5 rounded px-2 py-1 text-left text-sm hover:bg-base-200 {selected === entry.path ? 'bg-base-200 font-medium' : ''}" style:padding-left="{depth * 0.75 + 1.9}rem" onclick={() => { onopen(entry.path); }}>
          <Icon name="file-text" class="size-4 shrink-0 text-[var(--brand-text-tertiary)]" />
          <span class="min-w-0 flex-1 truncate">{entry.name}</span>
          {@const status = statusByPath.get(entry.path)}
          {#if status}<span class="badge badge-warning badge-xs" title={`${gitStatusLabel(status)}; index ${status.index}, working tree ${status.worktree}`} aria-label={`${gitStatusLabel(status)}; staged ${status.index !== '.'}, unstaged ${status.worktree !== '.'}`}>{code(status)}</span>{/if}
        </button>
      {/if}
    </li>
  {/each}
</ul>
