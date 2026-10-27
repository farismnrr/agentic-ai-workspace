<script lang="ts">
  import { onDestroy, untrack } from 'svelte';
  import type { Readable } from 'svelte/store';
  import ComposerBox from '../components/ComposerBox.svelte';
  import { attachmentsCanSend, readyPaths, type Attachment } from '../logic/attachments';
  import { findProject } from '../logic/derive';
  import { describeTaskError } from '../logic/task-errors';
  import { taskErrors } from '../content/tasks-copy';
  import { createController } from '../logic/task-actions';
  import { isExternalTask, originTaskHref, taskHref } from '../logic/task-links';
  import { submitTaskFollowUp } from '../logic/task-composer';
  import type { Task, TaskEvent } from '../logic/task-types';
  import { createThread } from '../logic/thread';
  import { projects } from '../logic/workspace-store';
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
  const views = $derived<{ id: View; label: string }[]>([{ id: 'chat', label: 'Chat' }, ...panelTabs]);
  const external = $derived($task !== undefined && isExternalTask($task.source, $task.workId ?? ''));
  const canWrite = $derived(external
    ? true
    : $task?.status === 'review' || $task?.status === 'done' || $task?.status === 'failed');
  let view = $state<View>('chat');
  let follow = $state('');
  let attachments = $state<Attachment[]>([]);
  let uploadError = $state('');
  let uploadBusy = $state(false);
  let uploadProgress = $state('');
  let linkedChatId = $state('');
  const archived = $derived($task?.archivedAt != null);
  const conversationId = $derived($task?.linkedTaskId ?? linkedChatId);
  const originHref = $derived($task === undefined ? undefined : originTaskHref($task));
  const panelTab = $derived<PanelTab>(view === 'chat' ? 'plan' : view);

  const controller = createController(id, thread);
  const error: Readable<string> = controller.error;

  async function sendFollowUp(text: string): Promise<void> {
    if (uploadBusy) return;
    uploadBusy = true;
    uploadError = '';
    if (!attachmentsCanSend(attachments)) {
      uploadBusy = false;
      return;
    }
    try {
      const paths = readyPaths(attachments);
      const result = await submitTaskFollowUp(text, paths, external, controller.follow, () => {
        follow = '';
        attachments = [];
      });
      if (result.linkedTaskId !== undefined) linkedChatId = result.linkedTaskId;
    } catch (failure) {
      uploadError = describeTaskError(failure);
    } finally {
      uploadBusy = false;
    }
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
      <TaskHeader task={$task} {archived} onstop={controller.stop} onretry={controller.retry} />
      {#if $task.originTaskId !== undefined}
        <div class="flex flex-wrap items-center gap-x-2 border-b border-[var(--brand-border)] bg-base-100 px-4 py-2 text-sm" role="status">
          <span class="badge badge-primary badge-soft badge-sm">Internal orchestrator</span>
          <span class="text-[var(--brand-text-secondary)]">Conversation started from an external task.</span>
          <a class="link link-hover font-medium" href={originHref}>View external task</a>
        </div>
      {/if}
      {#if external && conversationId !== ''}
        <div class="flex flex-wrap items-center gap-x-2 border-b border-[var(--brand-border)] bg-base-100 px-4 py-2 text-sm" role="status">
          <span>Your internal orchestrator conversation is ready.</span>
          <a class="link link-hover font-medium" href={taskHref($task.projectId, conversationId)}>Open conversation</a>
        </div>
      {/if}
      {#if archived}
        <div class="border-b border-[var(--brand-border)] bg-base-100 px-4 py-2" role="status">
          <span class="font-medium">This task is archived</span>
        </div>
      {/if}
      <div role="tablist" class="tabs tabs-border overflow-x-auto border-b border-[var(--brand-border)] bg-base-100 px-2 lg:hidden" aria-label="Task sections">
        {#each views as item (item.id)}
          <button role="tab" class="tab px-2.5 {view === item.id ? 'tab-active font-medium' : ''}" aria-selected={view === item.id} onclick={() => (view = item.id)}>{item.label}</button>
        {/each}
      </div>
      <div class="flex min-h-0 flex-1">
        <section class="min-w-0 flex-1 flex-col lg:flex {view === 'chat' ? 'flex' : 'hidden'}" aria-label="Conversation">
          <TaskThread task={$task} events={$events} draft={$draft} actions={controller.actions} onretry={$task.source === '' ? controller.retry : undefined} readOnly={archived} />
          {#if external && !archived}
            <p class="mx-auto w-full max-w-[51rem] px-4 pb-2 text-xs text-[var(--brand-text-secondary)] sm:px-6">
              Messages here go to the internal orchestrator. The external assistant will not reply here.
            </p>
          {/if}
          {#if !archived && $task.status === 'review'}
            <ReviewBar proven={$task.proof?.proven ?? false} onaccept={controller.accept} onchanges={focusFollowUp} />
          {/if}
          <div class="mx-auto w-full max-w-[51rem] px-4 pb-3 sm:px-6">
            <ComposerBox
              id="follow-up"
              placeholder={archived
                ? 'This task is archived and read-only.'
                : external
                  ? 'The external assistant will not reply here; the internal orchestrator will.'
                  : canWrite
                    ? 'Ask for a change or add details…'
                    : 'You can write again once the AI has finished or stopped.'}
              bind:text={follow}
              disabled={archived || !canWrite}
              busy={uploadBusy}
              {uploadProgress}
              error={uploadError || ($error === taskErrors['no_model'] ? '' : $error)}
              bind:attachments
              {projectId}
              attachable
              onsend={(text: string) => void sendFollowUp(text)}
            />
            {#if $error === taskErrors['no_model']}
              <p class="mt-2 text-sm text-error" role="alert">
                No AI model is ready for this conversation. Choose a provider and model in
                <a class="link link-hover font-medium" href="/app/settings/models">AI model settings</a>, then send again.
              </p>
            {/if}
          </div>
        </section>
        <aside class="min-w-0 flex-col bg-base-100 lg:flex lg:w-[26rem] lg:flex-none lg:border-l lg:border-[var(--brand-border)] {view === 'chat' ? 'hidden' : 'flex flex-1'}" aria-label="Work panel">
          <WorkPanel task={$task} events={$events} tab={panelTab} onselect={(tab: PanelTab) => (view = tab)} onundo={controller.actions.undo} />
        </aside>
      </div>
    </div>
  {/if}
</AppShell>
