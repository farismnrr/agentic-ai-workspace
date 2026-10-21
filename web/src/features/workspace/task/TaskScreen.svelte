<script lang="ts">
  import { onDestroy, untrack } from 'svelte';
  import type { Readable } from 'svelte/store';
  import ComposerBox from '../components/ComposerBox.svelte';
  import { attachmentsCanSend, readyPaths, withAttachments, type Attachment } from '../logic/attachments';
  import { findProject } from '../logic/derive';
  import { describeTaskError } from '../logic/task-errors';
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
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import { changeTasks, type TaskBatchAction } from '../logic/tasks-api';
  import { navigate } from '../../../app/router';
  import { refresh, removeTasks } from '../logic/workspace-store';

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
  let attachments = $state<Attachment[]>([]);
  let uploadError = $state('');
  let uploadBusy = $state(false);
  let uploadProgress = $state('');
  let confirming = $state<'archive' | 'delete' | undefined>();
  let taskActionError = $state('');
  let taskActionBusy = $state(false);
  const archived = $derived($task?.archivedAt != null);
  const running = $derived(['queued', 'planning', 'working'].includes($task?.status ?? ''));
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
      if (await controller.follow(withAttachments(text, paths))) {
        follow = '';
        attachments = [];
      }
    } catch (failure) {
      uploadError = describeTaskError(failure);
    } finally {
      uploadBusy = false;
    }
  }

  const focusFollowUp = (): void => document.getElementById('follow-up')?.focus();

  function requestTaskAction(action: 'archive' | 'delete'): void {
    if (action === 'delete' || running) confirming = action;
    else void applyTaskAction(action);
  }

  async function applyTaskAction(action: TaskBatchAction): Promise<void> {
    confirming = undefined;
    taskActionBusy = true;
    taskActionError = '';
    try {
      const [result] = await changeTasks(action, [id]);
      if (result?.ok !== true) throw new Error('Task action failed');
      if (action === 'archive' || action === 'delete') {
        removeTasks([id]);
        navigate('/app/tasks');
      } else {
        await Promise.all([thread.refresh(), refresh()]);
      }
    } catch {
      taskActionError = 'The task could not be updated. Try again.';
    } finally {
      taskActionBusy = false;
    }
  }
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
      {#if archived}
        <div class="flex flex-wrap items-center gap-3 border-b border-[var(--brand-border)] bg-base-100 px-4 py-2" role="status">
          <span class="font-medium">This task is archived</span>
          <button class="btn btn-outline btn-sm" type="button" disabled={taskActionBusy} onclick={() => void applyTaskAction('restore')}>Restore</button>
          <button class="btn btn-error btn-outline btn-sm" type="button" disabled={taskActionBusy} onclick={() => { requestTaskAction('delete'); }}>Delete permanently</button>
        </div>
      {:else}
        <div class="flex items-center gap-2 border-b border-[var(--brand-border)] bg-base-100 px-4 py-2">
          <button class="btn btn-outline btn-sm" type="button" disabled={taskActionBusy} onclick={() => { requestTaskAction('archive'); }}>Archive</button>
          <button class="btn btn-error btn-outline btn-sm" type="button" disabled={taskActionBusy} onclick={() => { requestTaskAction('delete'); }}>Delete permanently</button>
        </div>
      {/if}
      <p class="sr-only" aria-live="polite">{taskActionError}</p>
      <div role="tablist" class="tabs tabs-border overflow-x-auto border-b border-[var(--brand-border)] bg-base-100 px-2 lg:hidden" aria-label="Task sections">
        {#each views as item (item.id)}
          <button role="tab" class="tab px-2.5 {view === item.id ? 'tab-active font-medium' : ''}" aria-selected={view === item.id} onclick={() => (view = item.id)}>{item.label}</button>
        {/each}
      </div>
      <div class="flex min-h-0 flex-1">
        <section class="min-w-0 flex-1 flex-col lg:flex {view === 'chat' ? 'flex' : 'hidden'}" aria-label="Conversation">
          <TaskThread task={$task} events={$events} draft={$draft} actions={controller.actions} onretry={controller.retry} onproof={() => (view = 'proof')} readOnly={archived} />
          {#if !archived && $task.status === 'review'}
            <ReviewBar proven={$task.proof?.proven ?? false} onaccept={controller.accept} onchanges={focusFollowUp} />
          {/if}
          <div class="mx-auto w-full max-w-[51rem] px-4 pb-3 sm:px-6">
            <ComposerBox
              id="follow-up"
              placeholder={archived ? 'This task is archived and read-only.' : canWrite ? 'Ask for a change or add details…' : 'You can write again once the AI has finished or stopped.'}
              bind:text={follow}
              disabled={archived || !canWrite}
              busy={uploadBusy}
              {uploadProgress}
              error={uploadError || $error}
              bind:attachments
              {projectId}
              attachable
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

<ConfirmModal open={confirming !== undefined}
  title={confirming === 'delete' ? 'Delete 1 task permanently?' : 'Archive this task?'}
  body={confirming === 'delete' ? 'This cannot be undone.' : 'The task will move to Archive.'}
  warning={confirming === 'archive' && running ? '1 task is still running and will be stopped.' : ''}
  confirmLabel={confirming === 'delete' ? 'Delete permanently' : 'Archive'} cancelLabel="Cancel"
  oncancel={() => (confirming = undefined)} onconfirm={() => void applyTaskAction(confirming ?? 'archive')} />
