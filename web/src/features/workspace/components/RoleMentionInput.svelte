<script lang="ts">
  import { tick } from 'svelte';
  import { mentionQuery, insertMention } from '../logic/role-mentions';

  interface Props {
    id: string;
    text?: string;
    placeholder: string;
    disabled: boolean;
    onkeydown: (event: KeyboardEvent) => void;
    onpaste: (event: ClipboardEvent) => void;
  }
  let { id, text = $bindable(), placeholder, disabled, onkeydown, onpaste }: Props = $props();
  let caret = $state(0);
  let dismissed = $state(false);
  let input: HTMLTextAreaElement;
  const query = $derived(mentionQuery(text ?? '', caret));
  const visible = $derived(!dismissed && query !== null && 'deep-review'.includes(query.toLowerCase()));

  function track(): void { caret = input.selectionStart; dismissed = false; }

  async function choose(): Promise<void> {
    const result = insertMention(text ?? '', caret, 'deep-review');
    text = result.text;
    caret = result.caret;
    dismissed = true;
    await tick();
    input.focus();
    input.setSelectionRange(caret, caret);
  }

  function keys(event: KeyboardEvent): void {
    if (visible && !event.isComposing) {
      if (event.key === 'Escape') { dismissed = true; event.preventDefault(); return; }
      if (event.key === 'Enter' || event.key === 'Tab') {
        event.preventDefault();
        void choose();
        return;
      }
    }
    onkeydown(event);
  }
</script>

{#if visible}
  <div id="{id}-mentions" role="listbox" aria-label="Workflow modifiers"
    class="absolute bottom-full left-2 right-2 z-30 mb-2 rounded-xl border border-base-300 bg-base-100 p-1 shadow-xl">
    <button type="button" role="option" aria-selected="true"
      class="flex w-full items-center gap-2 rounded-lg bg-base-200 px-3 py-2 text-left text-sm hover:bg-base-300"
      onmousedown={(event) => { event.preventDefault(); }} onclick={() => { void choose(); }}>
      <span class="font-medium">@deep-review</span>
      <span class="text-xs text-base-content/60">Full review and relevant regression checks</span>
    </button>
  </div>
{/if}
<textarea {id} bind:this={input} bind:value={text} {placeholder} {disabled}
  onkeydown={keys} {onpaste} oninput={track} onkeyup={track} onclick={track}
  aria-autocomplete="list" aria-controls={visible ? `${id}-mentions` : undefined}
  rows="2"
  class="field-sizing-content max-h-52 min-h-14 w-full resize-none bg-transparent
    px-4 pb-1 pt-3.5 text-sm focus:outline-none"></textarea>
