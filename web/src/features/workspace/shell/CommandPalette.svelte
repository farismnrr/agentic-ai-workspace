<script lang="ts">
  import { navigate } from '../../../app/router';
  import Icon from '../../../shared/ui/Icon.svelte';
  import { projects, tasks } from '../logic/workspace-store';
  import { allCommands, filterCommands, type Command } from './commands';
  import { paletteOpen } from './shell-state';

  let dialog = $state<HTMLDialogElement>();
  let query = $state('');
  let active = $state(0);
  const results = $derived(filterCommands(allCommands($projects, $tasks), query));

  $effect(() => {
    if ($paletteOpen && dialog?.open !== true) dialog?.showModal();
    if (!$paletteOpen && dialog?.open === true) dialog.close();
  });

  function choose(command: Command | undefined): void {
    if (command === undefined) return;
    paletteOpen.set(false);
    navigate(command.href);
  }

  function onkeydown(event: KeyboardEvent): void {
    if (event.key === 'ArrowDown') active = Math.min(active + 1, results.length - 1);
    else if (event.key === 'ArrowUp') active = Math.max(active - 1, 0);
    else if (event.key === 'Enter') choose(results[active]);
    else return;
    event.preventDefault();
  }

  function onclose(): void {
    paletteOpen.set(false);
    query = '';
    active = 0;
  }
</script>

<dialog bind:this={dialog} class="modal modal-top items-start" {onclose} aria-label="Search and jump">
  <div class="modal-box mx-auto mt-[12vh] max-w-xl p-0">
    <label class="flex items-center gap-3 border-b border-[var(--brand-border)] px-4">
      <Icon name="search" class="size-4 opacity-60" />
      <input
        class="h-10 flex-1 bg-transparent focus:outline-none"
        type="text"
        placeholder="Search tasks, projects, pages…"
        aria-label="Search tasks, projects, pages"
        bind:value={query}
        oninput={() => (active = 0)}
        {onkeydown}
      />
      <kbd class="kbd kbd-sm">Esc</kbd>
    </label>
    <ul class="menu w-full p-2" aria-label="Results">
      {#each results as item, index (item.id)}
        <li>
          <button class="flex items-center gap-3 {index === active ? 'bg-base-200!' : ''}" type="button" onclick={() => { choose(item); }} onmouseenter={() => (active = index)}>
            <Icon name={item.icon} class="size-4" /><span class="flex-1 text-left">{item.label}</span>
            <span class="text-xs text-[var(--brand-text-tertiary)]">{item.hint}</span>
          </button>
        </li>
      {:else}
        <li class="px-4 py-3 text-[var(--brand-text-secondary)]">Nothing matches “{query}”.</li>
      {/each}
    </ul>
  </div>
  <form method="dialog" class="modal-backdrop"><button aria-label="Close search">close</button></form>
</dialog>
