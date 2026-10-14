<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import { clearChat, loadChat, sendChatMessage, type ChatMessage } from '../logic/chat-api';
  import { describeTaskError } from '../logic/task-errors';
  import EscalateModal from './EscalateModal.svelte';

  interface Props {
    projectId: string;
  }

  const { projectId }: Props = $props();

  let messages = $state<ChatMessage[]>([]);
  let input = $state('');
  let loading = $state(true);
  let sending = $state(false);
  let error = $state('');
  let escalateOpen = $state(false);
  let escalatePrompt = $state('');

  async function fetchMessages(): Promise<void> {
    loading = true;
    error = '';
    try {
      const data = await loadChat(projectId);
      messages = data.messages;
    } catch (failure) {
      error = describeTaskError(failure);
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    if (projectId !== '') void fetchMessages();
  });

  async function onsend(): Promise<void> {
    const text = input.trim();
    if (text === '' || sending) return;
    sending = true;
    error = '';
    input = '';
    try {
      const res = await sendChatMessage(projectId, text);
      messages = [...messages, res.userMessage, res.assistantMessage];
    } catch (failure) {
      error = describeTaskError(failure);
    } finally {
      sending = false;
    }
  }

  function onkeydown(event: KeyboardEvent): void {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault();
      void onsend();
    }
  }

  async function onclear(): Promise<void> {
    if (sending) return;
    try {
      await clearChat(projectId);
      messages = [];
    } catch (failure) {
      error = describeTaskError(failure);
    }
  }

  function triggerEscalate(prefill: string): void {
    escalatePrompt = prefill;
    escalateOpen = true;
  }
</script>

<div class="flex flex-col gap-4">
  <div class="flex items-center justify-between border-b border-[var(--brand-border)] pb-2 text-xs text-[var(--brand-text-secondary)]">
    <span>Direct discussion & Q&A. No plan or file changes will be made here.</span>
    {#if messages.length > 0}
      <button class="btn btn-ghost btn-xs text-error" type="button" onclick={onclear} disabled={sending}>
        <Icon name="trash" class="size-3.5" /> Clear chat
      </button>
    {/if}
  </div>

  {#if loading}
    <div class="flex justify-center py-12"><span class="loading loading-spinner loading-md text-primary"></span></div>
  {:else if messages.length === 0}
    <div class="rounded-2xl border border-dashed border-[var(--brand-border-strong)] p-8 text-center">
      <div class="mx-auto flex size-10 items-center justify-center rounded-full bg-base-200 text-primary">
        <Icon name="chat" class="size-5" />
      </div>
      <h3 class="mt-3 text-base font-semibold">Chat with Awam</h3>
      <p class="mx-auto mt-1 max-w-sm text-sm text-[var(--brand-text-secondary)]">
        Ask questions, brainstorm features, or explore solutions. When ready, escalate to an agentic Task.
      </p>
    </div>
  {:else}
    <div class="flex flex-col gap-3 max-h-[500px] overflow-y-auto px-1">
      {#each messages as msg (msg.id)}
        {#if msg.role === 'user'}
          <div class="chat chat-end">
            <div class="chat-header text-xs opacity-60">You</div>
            <div class="chat-bubble chat-bubble-primary text-sm whitespace-pre-wrap">{msg.content}</div>
          </div>
        {:else}
          <div class="chat chat-start">
            <div class="chat-header text-xs opacity-60">Awam</div>
            <div class="chat-bubble border border-[var(--brand-border)] bg-base-200 text-base-content text-sm whitespace-pre-wrap">
              {msg.content}
            </div>
            <div class="chat-footer mt-1 flex gap-2">
              <button
                class="btn btn-outline btn-xs gap-1 rounded-full text-primary"
                type="button"
                onclick={() => { triggerEscalate(msg.content); }}
              >
                <Icon name="arrow-right" class="size-3" /> Turn into Task
              </button>
            </div>
          </div>
        {/if}
      {/each}
      {#if sending}
        <div class="chat chat-start">
          <div class="chat-header text-xs opacity-60">Awam</div>
          <div class="chat-bubble border border-[var(--brand-border)] bg-base-200 text-base-content text-sm">
            <span class="loading loading-dots loading-xs"></span>
          </div>
        </div>
      {/if}
    </div>
  {/if}

  {#if error !== ''}<p class="text-xs text-error">{error}</p>{/if}

  <div class="relative mt-2">
    <textarea
      bind:value={input}
      {onkeydown}
      rows="2"
      placeholder="Type a message or question… (Press Enter to send)"
      disabled={sending}
      class="textarea textarea-bordered w-full resize-none text-sm pr-12"
    ></textarea>
    <button
      class="btn btn-primary btn-circle btn-sm absolute bottom-3 right-3"
      type="button"
      onclick={onsend}
      disabled={sending || input.trim() === ''}
      aria-label="Send message"
    >
      <Icon name="arrow-right" class="size-4" />
    </button>
  </div>
</div>

<EscalateModal {projectId} open={escalateOpen} initialPrompt={escalatePrompt} onclose={() => { escalateOpen = false; }} />
