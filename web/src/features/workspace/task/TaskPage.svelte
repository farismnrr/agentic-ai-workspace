<script lang="ts">
  import ComposerBox from '../components/ComposerBox.svelte';
  import { findProject } from '../content/projects';
  import { findTask } from '../content/tasks';
  import AppShell from '../shell/AppShell.svelte';
  import WorkPanel from './panel/WorkPanel.svelte';
  import { panelTabs, type PanelTab } from './panel/tabs';
  import ReviewBar from './ReviewBar.svelte';
  import TaskHeader from './TaskHeader.svelte';
  import TaskThread from './thread/TaskThread.svelte';

  interface Props {
    projectId: string;
    taskId: string;
  }

  type View = 'chat' | PanelTab;

  const { projectId, taskId }: Props = $props();
  const project = $derived(findProject(projectId));
  const task = $derived(findTask(projectId, taskId));
  const crumbs = $derived([{ label: 'Projects', href: '/app/projects' }, { label: project?.name ?? 'Project', href: `/app/projects/${projectId}` }]);
  const views: { id: View; label: string }[] = [{ id: 'chat', label: 'Chat' }, ...panelTabs];
  let view = $state<View>('chat');
  const panelTab = $derived<PanelTab>(view === 'chat' ? 'plan' : view);

  function select(tab: PanelTab): void {
    view = tab;
  }
</script>

<svelte:head><title>{task?.title ?? 'Task'} · Masih Awam</title></svelte:head>

<AppShell {crumbs} width="split">
  {#if task === undefined}
    <div class="p-8"><h1 class="text-2xl">We couldn't find that task</h1><a class="link link-hover mt-4 inline-block font-medium text-primary" href="/app/projects/{projectId}">Back to the project</a></div>
  {:else}
    <div class="flex flex-1 flex-col lg:h-[calc(100dvh-3rem)]">
      <TaskHeader {task} />
      <div role="tablist" class="tabs tabs-border overflow-x-auto border-b border-[var(--brand-border)] bg-base-100 px-2 lg:hidden" aria-label="Task sections">
        {#each views as item (item.id)}
          <button role="tab" class="tab px-2.5 {view === item.id ? 'tab-active font-medium' : ''}" aria-selected={view === item.id} onclick={() => (view = item.id)}>{item.label}</button>
        {/each}
      </div>
      <div class="flex min-h-0 flex-1">
        <section class="min-w-0 flex-1 flex-col lg:flex {view === 'chat' ? 'flex' : 'hidden'}" aria-label="Conversation">
          <TaskThread {task} />
          {#if task.status === 'review'}<ReviewBar />{/if}
          <div class="mx-auto w-full max-w-[51rem] px-4 pb-3 sm:px-6"><ComposerBox id="follow-up" placeholder="Ask for a change or add details…" /></div>
        </section>
        <aside class="min-w-0 flex-col bg-base-100 lg:flex lg:w-[26rem] lg:flex-none lg:border-l lg:border-[var(--brand-border)] {view === 'chat' ? 'hidden' : 'flex flex-1'}" aria-label="Work panel">
          <WorkPanel {task} site={project?.site ?? ''} tab={panelTab} onselect={select} />
        </aside>
      </div>
    </div>
  {/if}
</AppShell>
