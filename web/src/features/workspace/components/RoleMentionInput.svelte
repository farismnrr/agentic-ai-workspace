<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { loadAgentTeam, type TeamRole } from '../logic/agent-team-api';
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
  let roles = $state<TeamRole[]>([]);
  let caret = $state(0);
  let selected = $state(0);
  let input: HTMLTextAreaElement;
  const query = $derived(mentionQuery(text ?? "", caret));
  const matches = $derived(query === null ? [] : roles.filter((role) =>
    role.enabled && !role.lead && (role.key.toLowerCase().includes(query.toLowerCase()) ||
      role.name.toLowerCase().includes(query.toLowerCase()))).slice(0, 8));

  onMount(() => {
    let active = true;
    void loadAgentTeam().then((team) => { if (active) roles = team.roles; }).catch(() => { roles = []; });
    return () => { active = false; };
  });

  function track(): void {
    caret = input.selectionStart;
    selected = 0;
  }

  async function choose(role: TeamRole): Promise<void> {
    const result = insertMention(text ?? "", caret, role.key);
    text = result.text;
    caret = result.caret;
    await tick();
    input.focus();
    input.setSelectionRange(caret, caret);
  }

  function keys(event: KeyboardEvent): void {
    if (matches.length > 0 && !event.isComposing && handleMentionKey(event)) return;
    onkeydown(event);
  }

  function handleMentionKey(event: KeyboardEvent): boolean {
    if (event.key === 'Escape') { caret = -1; event.preventDefault(); return true; }
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      selected = (selected + (event.key === 'ArrowDown' ? 1 : -1) + matches.length) % matches.length;
      event.preventDefault();
      return true;
    }
    if (event.key === 'Enter' || event.key === 'Tab') {
      const role = matches[selected] ?? matches[0];
      if (role !== undefined) { event.preventDefault(); void choose(role); return true; }
    }
    return false;
  }

  function releaseKey(event: KeyboardEvent): void {
    if (event.key === 'ArrowLeft' || event.key === 'ArrowRight') track();
  }
</script>

{#if matches.length > 0}
  <div id="{id}-mentions" role="listbox" aria-label="Mention a teammate"
    class="absolute bottom-full left-2 right-2 z-30 mb-2 max-h-64 overflow-auto rounded-xl border border-base-300 bg-base-100 p-1 shadow-xl">
    {#each matches as role, index (role.key)}
      <button type="button" role="option" aria-selected={index === selected}
        class="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-sm hover:bg-base-200 {index === selected ? 'bg-base-200' : ''}"
        onmousedown={(event) => { event.preventDefault(); }} onclick={() => { void choose(role); }}>
        <span class="font-medium">@{role.key}</span>
        <span class="truncate text-xs text-base-content/60">{role.name}</span>
      </button>
    {/each}
  </div>
{/if}
<textarea {id} bind:this={input} bind:value={text} {placeholder} {disabled}
  onkeydown={keys} {onpaste} oninput={track} onkeyup={releaseKey} onclick={track}
  aria-autocomplete="list" aria-controls={matches.length > 0 ? `${id}-mentions` : undefined}
  rows="2"
  class="field-sizing-content max-h-52 min-h-14 w-full resize-none bg-transparent
    px-4 pb-1 pt-3.5 text-sm focus:outline-none"></textarea>
