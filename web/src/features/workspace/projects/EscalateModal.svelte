<script lang="ts">
  import { navigate } from '../../../app/router';
  import { askLevels } from '../content/tasks-copy';
  import { escalateChatToTask } from '../logic/chat-api';
  import { describeTaskError } from '../logic/task-errors';
  import type { ApprovalMode } from '../logic/task-types';
  import { refresh } from '../logic/workspace-store';

  interface Props {
    projectId: string;
    open?: boolean | undefined;
    initialPrompt?: string | undefined;
    onclose: () => void;
  }

  let { projectId, open = false, initialPrompt = '', onclose }: Props = $props();

  let prompt = $state('');
  let mode: ApprovalMode = $state('risky');
  let busy = $state(false);
  let error = $state('');

  $effect(() => {
    if (open) {
      prompt = initialPrompt;
      error = '';
    }
  });

  async function onsubmit(event: SubmitEvent): Promise<void> {
    event.preventDefault();
    if (busy) return;
    busy = true;
    error = '';
    try {
      const res = await escalateChatToTask(projectId, prompt.trim(), mode);
      await refresh();
      onclose();
      navigate(`/app/projects/${projectId}/tasks/${res.taskId}`);
    } catch (failure) {
      error = describeTaskError(failure);
    } finally {
      busy = false;
    }
  }
</script>

{#if open}
  <dialog class="modal modal-open">
    <div class="modal-box max-w-lg">
      <h3 class="text-lg font-bold">Turn into Task</h3>
      <p class="py-2 text-sm text-[var(--brand-text-secondary)]">
        Convert your chat discussion into an agentic task with a plan and verifiable proof.
      </p>
      <form {onsubmit} class="mt-2 space-y-4">
        <div>
          <label class="label text-xs font-medium" for="escalate-prompt">Task Instruction</label>
          <textarea
            id="escalate-prompt"
            bind:value={prompt}
            rows="3"
            placeholder="What should the agent do?"
            class="textarea textarea-bordered w-full text-sm"
            required
            disabled={busy}
          ></textarea>
        </div>
        <div>
          <label class="label text-xs font-medium" for="escalate-mode">Approval Mode</label>
          <select id="escalate-mode" bind:value={mode} class="select select-bordered select-sm w-full" disabled={busy}>
            {#each askLevels as level (level.value)}
              <option value={level.value}>{level.label}</option>
            {/each}
          </select>
        </div>
        {#if error !== ''}<p class="text-xs text-error">{error}</p>{/if}
        <div class="modal-action">
          <button class="btn btn-ghost btn-sm" type="button" onclick={onclose} disabled={busy}>Cancel</button>
          <button class="btn btn-primary btn-sm" type="submit" disabled={busy || prompt.trim() === ''}>
            {#if busy}<span class="loading loading-spinner loading-xs"></span>{/if}
            Start Task
          </button>
        </div>
      </form>
    </div>
    <form method="dialog" class="modal-backdrop">
      <button type="button" onclick={onclose}>close</button>
    </form>
  </dialog>
{/if}
