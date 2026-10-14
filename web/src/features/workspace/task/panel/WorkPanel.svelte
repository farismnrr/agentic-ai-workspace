<script lang="ts">
  import type { Task } from '../../content/task-types';
  import ChangesTab from './ChangesTab.svelte';
  import FilesTab from './FilesTab.svelte';
  import PlanTab from './PlanTab.svelte';
  import PreviewTab from './PreviewTab.svelte';
  import ProofTab from './ProofTab.svelte';
  import { panelTabs, type PanelTab } from './tabs';

  interface Props {
    task: Task;
    site: string;
    tab: PanelTab;
    onselect: (tab: PanelTab) => void;
  }

  const { task, site, tab, onselect }: Props = $props();
</script>

<div role="tablist" class="tabs tabs-border hidden border-b border-[var(--brand-border)] px-3 lg:flex" aria-label="Work panel">
  {#each panelTabs as item (item.id)}
    <button role="tab" class="tab px-2.5 {tab === item.id ? 'tab-active font-medium' : ''}" aria-selected={tab === item.id} onclick={() => { onselect(item.id); }}>{item.label}</button>
  {/each}
</div>
<div class="min-h-0 flex-1 overflow-y-auto p-4" role="tabpanel" aria-label={tab}>
  {#if tab === 'plan'}<PlanTab plan={task.plan} />
  {:else if tab === 'proof'}<ProofTab proof={task.proof} />
  {:else if tab === 'changes'}<ChangesTab changes={task.changes} />
  {:else if tab === 'preview'}<PreviewTab {task} {site} />
  {:else}<FilesTab changes={task.changes} />{/if}
</div>
