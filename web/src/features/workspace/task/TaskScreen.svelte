<script lang="ts">
  import { onDestroy, untrack } from 'svelte';
  import type { Readable } from 'svelte/store';
  import ComposerBox from '../components/ComposerBox.svelte';
  import { findProject } from '../logic/derive';
  import { createController } from '../logic/task-actions';
  import type { Task, TaskEvent } from '../logic/task-types';
  import { createThread } from '../logic/thread';
  import { projects } from '../logic/workspace-store';
  import AppShell from '../shell/AppShell.svelte';
  import WorkPanel from './panel/WorkPanel.svelte';
  import { panelTabsFor, type PanelTab } from './panel/tabs';
  import ReviewBar from './ReviewBar.svelte';
  import TaskHeader from './TaskHeader.svelte';
  import TaskThread from './thread/TaskThread.svelte';

  interface Props {
    projectId: string;
    taskId: string;
  }

  type View = 'chat' | PanelTab;

  const { projectId, taskId }: Props = $props();
  // TaskPage keys this component by task, so the thread is created once per task.
  const id = untrack(() => taskId);
  const thread = createThread(id);
  const task: Readable<Task | undefined> = thread.task;
  const events: Readable<TaskEvent[]> = thread.events;
  const missing: Readable<boolean> = thread.missing;
  const draft: Readable<string> = thread.draft;
  onDestroy(thread.stop);

  const project = $derived(findProject($projects, projectId));
  const crumbs = $derived([{ label: 'Projects', href: '/app/projects' }, { label: project?.name ?? 'Project', href: `/app/projects/${projectId}` }]);
  const views = $derived<{ id: View; label: string }[]>([{ id: 'chat', label: 'Chat' }, ...panelTabsFor($task?.team ?? false)]);
  const canWrite = $derived($task?.status === 'review' || $task?.status === 'done' || $task?.status === 'failed');
  let view = $state<View>('chat');
  let follow = $state('');
  const panelTab = $derived<PanelTab>(view === 'chat' ? 'plan' : view);

  const controller = createController(id, thread);
  const error: Readable<string> = controller.error;

  async function sendFollowUp(text: string): Promise<void> {
    if (await controller.follow(text)) follow = '';
  }

  const focusFollowUp = (): void => document.getElementById('follow-up')?.focus();
</script>

<svelte:head><title>{$task?.title ?? 'Task'} · Masih Awam</title></svelte:head>

<AppShell {crumbs} width="split">
  {#if $task === undefined}
    <div class="p-8">
      {#if $missing}
        <h1 class="text-2xl">We couldn't find that task</h1>
        <a class="link link-hover mt-4 inline-block font-medium text-primary" href="/app/projects/{projectId}">Back to the project</a>
      {:else}
        <span class="loading loading-spinner" aria-label="Loading"></span>
      {/if}
    </div>
  {:else}
    <div class="flex h-[calc(100dvh-3rem)] flex-col">
      <TaskHeader task={$task} onstop={controller.stop} onretry={controller.retry} />
      <div role="tablist" class="tabs tabs-border overflow-x-auto border-b border-[var(--brand-border)] bg-base-100 px-2 lg:hidden" aria-label="Task sections">
        {#each views as item (item.id)}
          <button role="tab" class="tab px-2.5 {view === item.id ? 'tab-active font-medium' : ''}" aria-selected={view === item.id} onclick={() => (view = item.id)}>{item.label}</button>
        {/each}
      </div>
      <div class="flex min-h-0 flex-1">
        <section class="min-w-0 flex-1 flex-col lg:flex {view === 'chat' ? 'flex' : 'hidden'}" aria-label="Conversation">
          <TaskThread task={$task} events={$events} draft={$draft} actions={controller.actions} onretry={controller.retry} onproof={() => (view = 'proof')} />
          {#if $task.status === 'review'}
            <ReviewBar proven={$task.proof?.proven ?? false} onaccept={controller.accept} onchanges={focusFollowUp} />
          {/if}
          <div class="mx-auto w-full max-w-[51rem] px-4 pb-3 sm:px-6">
            <ComposerBox
              id="follow-up"
              placeholder={canWrite ? 'Ask for a change or add details…' : 'You can write again once the AI has finished or stopped.'}
              bind:text={follow}
              disabled={!canWrite}
              error={$error}
              onsend={(text: string) => void sendFollowUp(text)}
            />
          </div>
        </section>
        <aside class="min-w-0 flex-col bg-base-100 lg:flex lg:w-[26rem] lg:flex-none lg:border-l lg:border-[var(--brand-border)] {view === 'chat' ? 'hidden' : 'flex flex-1'}" aria-label="Work panel">
          <WorkPanel task={$task} events={$events} tab={panelTab} onselect={(tab: PanelTab) => (view = tab)} onundo={controller.actions.undo} />
        </aside>
      </div>
    </div>
  {/if}
</AppShell>
