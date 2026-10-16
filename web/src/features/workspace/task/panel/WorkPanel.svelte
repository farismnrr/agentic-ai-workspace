<script lang="ts">
  import { changesOf } from '../../logic/derive';
  import type { Task, TaskEvent } from '../../logic/task-types';
  import ChangesTab from './ChangesTab.svelte';
  import FilesTab from './FilesTab.svelte';
  import PlanTab from './PlanTab.svelte';
  import ProofTab from './ProofTab.svelte';
  import TeamTab from './TeamTab.svelte';
  import { panelTabsFor, type PanelTab } from './tabs';

  interface Props {
    task: Task;
    events: TaskEvent[];
    tab: PanelTab;
    onselect: (tab: PanelTab) => void;
    onundo: (eventId: string) => void;
  }

  const { task, events, tab, onselect, onundo }: Props = $props();
  const changes = $derived(changesOf(events));
  const tabs = $derived(panelTabsFor(task.team));
</script>

<div role="tablist" class="tabs tabs-border hidden border-b border-[var(--brand-border)] px-3 lg:flex" aria-label="Work panel">
  {#each tabs as item (item.id)}
    <button role="tab" class="tab px-2.5 {tab === item.id ? 'tab-active font-medium' : ''}" aria-selected={tab === item.id} onclick={() => { onselect(item.id); }}>{item.label}</button>
  {/each}
</div>
<div class="min-h-0 flex-1 overflow-x-hidden overflow-y-auto p-4 wrap-anywhere" role="tabpanel" aria-label={tab}>
  {#if tab === 'plan'}<PlanTab plan={task.plan} prd={task.prd} />
  {:else if tab === 'team'}<TeamTab {events} waiting={task.status === 'needs-input'} />
  {:else if tab === 'proof'}<ProofTab proof={task.proof} />
  {:else if tab === 'changes'}<ChangesTab {changes} {onundo} />
  {:else}<FilesTab {changes} />{/if}
</div>
