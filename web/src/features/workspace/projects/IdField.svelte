<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import { newProjectCopy } from '../content/templates-copy';
  import { checkProjectId } from '../logic/projects-api';

  interface Props {
    /** The name being typed; the ID follows it until the person edits the ID. */
    name: string;
    slug?: string | undefined;
    /** Told whether the ID is not usable yet (being checked, taken or invalid), so the form should not be sent. */
    onblock: (blocked: boolean) => void;
  }

  let { name, slug = $bindable(''), onblock }: Props = $props();

  let editing = $state(false);
  let status = $state<'idle' | 'checking' | 'free' | 'taken' | 'invalid'>('idle');
  let suggestion = $state('');
  let asked = 0;

  $effect(() => { onblock(name.trim() !== '' && status !== 'free'); });

  // Follows the name, or checks the ID the person typed. A server that cannot answer is not trusted to say "free".
  $effect(() => {
    const typed = editing ? slug.trim() : '';
    const text = name.trim();
    if (!editing && text === '') { slug = ''; status = 'idle'; return; }
    status = 'checking';
    const turn = ++asked;
    const timer = setTimeout(() => {
      void checkProjectId(typed, text).then((answer) => {
        if (turn !== asked) return;
        suggestion = answer.suggestion;
        if (!editing) { slug = answer.suggestion; status = 'free'; return; }
        status = !answer.valid ? 'invalid' : answer.available ? 'free' : 'taken';
      }, () => { if (turn === asked) status = 'idle'; });
    }, 350);
    return () => { clearTimeout(timer); };
  });
</script>

<div class="mt-4">
  <label class="block font-medium" for="project-id">{newProjectCopy.idLabel}</label>
  {#if editing}
    <input id="project-id" class="input mt-1 w-full font-mono" class:input-error={status === 'taken' || status === 'invalid'} maxlength="30" aria-describedby="project-id-hint" aria-invalid={status === 'taken' || status === 'invalid'} value={slug} oninput={(event) => { slug = event.currentTarget.value.toLowerCase(); }} />
  {:else}
    <p class="mt-1 flex items-center gap-2">
      <code class="rounded bg-base-200 px-2 py-1 font-mono text-sm" id="project-id">{slug === '' ? '—' : slug}</code>
      <button class="link link-primary text-sm" type="button" disabled={slug === ''} onclick={() => { editing = true; }}><Icon name="pencil" class="mr-1 inline size-4 align-text-bottom" />{newProjectCopy.idEdit}</button>
    </p>
  {/if}
  <p id="project-id-hint" class="mt-1 min-h-5 text-sm" aria-live="polite">
    {#if status === 'taken'}
      <span class="text-error"><Icon name="alert-triangle" class="mr-1 inline size-4 align-text-bottom" />{newProjectCopy.idTaken(slug)}</span>
      <button class="link link-primary ml-1" type="button" onclick={() => { slug = suggestion; }}>{newProjectCopy.useSuggestion(suggestion)}</button>
    {:else if status === 'invalid'}
      <span class="text-error"><Icon name="alert-triangle" class="mr-1 inline size-4 align-text-bottom" />{newProjectCopy.idInvalid}</span>
      <button class="link link-primary ml-1" type="button" onclick={() => { slug = suggestion; }}>{newProjectCopy.useSuggestion(suggestion)}</button>
    {:else if status === 'free'}
      <span class="text-success"><Icon name="check" class="mr-1 inline size-4 align-text-bottom" />{newProjectCopy.idFree(slug)}</span>
      <span class="block text-[var(--brand-text-tertiary)]">{newProjectCopy.idPermanent}</span>
    {:else if status === 'checking'}
      <span class="text-[var(--brand-text-tertiary)]">{newProjectCopy.idChecking}</span>
    {/if}
  </p>
</div>
