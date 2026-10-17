<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import type { FileEntry } from '../logic/files-api';
  import TreeLevel from './TreeLevel.svelte';

  interface Props {
    /** The entries of every folder read so far, by folder path. */
    dirs: Record<string, FileEntry[]>;
    expanded: ReadonlySet<string>;
    path: string;
    depth: number;
    selected: string | undefined;
    /** Paths this task changed; they get a mark. */
    changed: ReadonlySet<string>;
    ontoggle: (path: string) => void;
    onopen: (path: string) => void;
    ondeletefolder: (path: string) => void;
  }

  const { dirs, expanded, path, depth, selected, changed, ontoggle, onopen, ondeletefolder }: Props = $props();
  const entries = $derived(dirs[path] ?? []);
  const touches = (folder: string): boolean => [...changed].some((file) => file.startsWith(`${folder}/`));
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
            {#if touches(entry.path)}<span class="size-1.5 shrink-0 rounded-full bg-warning" title="Changed by this task"></span>{/if}
          </button>
          <button type="button" class="btn btn-ghost btn-xs mr-1 text-error opacity-0 focus-visible:opacity-100 group-hover:opacity-100" aria-label="Delete folder {entry.name}" onclick={() => { ondeletefolder(entry.path); }}><Icon name="trash" class="size-3.5" /></button>
        </div>
        {#if expanded.has(entry.path)}
          <TreeLevel {dirs} {expanded} path={entry.path} depth={depth + 1} {selected} {changed} {ontoggle} {onopen} {ondeletefolder} />
        {/if}
      {:else}
        <button type="button" class="flex w-full items-center gap-1.5 rounded px-2 py-1 text-left text-sm hover:bg-base-200 {selected === entry.path ? 'bg-base-200 font-medium' : ''}" style:padding-left="{depth * 0.75 + 1.9}rem" onclick={() => { onopen(entry.path); }}>
          <Icon name="file-text" class="size-4 shrink-0 text-[var(--brand-text-tertiary)]" />
          <span class="min-w-0 flex-1 truncate">{entry.name}</span>
          {#if changed.has(entry.path)}<span class="badge badge-warning badge-xs">changed</span>{/if}
        </button>
      {/if}
    </li>
  {/each}
</ul>
