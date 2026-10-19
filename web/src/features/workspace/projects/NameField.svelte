<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import { newProjectCopy } from '../content/templates-copy';
  import { randomProjectName } from '../logic/random-name';
  import { checkProjectName, type NameCheck } from '../logic/projects-api';

  interface Props {
    name?: string | undefined;
    /** Told whether the name is taken or still being checked, so the form should not be sent. */
    onblock: (blocked: boolean) => void;
  }

  let { name = $bindable(''), onblock }: Props = $props();

  let status = $state<'idle' | 'checking' | 'free' | 'taken'>('idle');
  let suggestion = $state('');
  let asked = 0;

  $effect(() => { onblock(name.trim() !== '' && status !== 'free'); });

  $effect(() => {
    const text = name.trim();
    if (text === '') { status = 'idle'; return; }
    status = 'checking';
    const turn = ++asked;
    const timer = setTimeout(() => {
      void checkProjectName(text).then((answer) => {
        if (turn !== asked) return;
        status = answer.available ? 'free' : 'taken';
        suggestion = answer.suggestion;
      }, () => { if (turn === asked) status = 'free'; });
    }, 350);
    return () => { clearTimeout(timer); };
  });

  /** Tries a few random names and keeps the first free one, so the dice never hands back a taken name. */
  async function roll(): Promise<void> {
    for (let attempt = 0; attempt < 5; attempt++) {
      const candidate = randomProjectName();
      const answer: NameCheck = await checkProjectName(candidate).catch((): NameCheck => ({ available: true, suggestion: candidate }));
      if (answer.available) { name = candidate; return; }
    }
    name = randomProjectName();
  }
</script>

<label class="mt-4 block font-medium" for="project-name">Name</label>
<div class="join mt-1 w-full">
  <input id="project-name" class="input join-item w-full" class:input-error={status === 'taken'} maxlength="80" placeholder="Bakery website" required aria-describedby="project-name-hint" aria-invalid={status === 'taken'} bind:value={name} />
  <button class="btn join-item btn-square" type="button" title={newProjectCopy.random} aria-label={newProjectCopy.random} onclick={() => void roll()}><Icon name="dice" class="size-5" /></button>
</div>
<p id="project-name-hint" class="mt-1 min-h-5 text-sm" aria-live="polite">
  {#if status === 'taken'}
    <span class="text-error"><Icon name="alert-triangle" class="mr-1 inline size-4 align-text-bottom" />{newProjectCopy.nameTaken(name.trim())}</span>
    <button class="link link-primary ml-1" type="button" onclick={() => { name = suggestion; }}>{newProjectCopy.useSuggestion(suggestion)}</button>
  {:else if status === 'free'}
    <span class="text-success"><Icon name="check" class="mr-1 inline size-4 align-text-bottom" />{newProjectCopy.nameFree(name.trim())}</span>
  {:else}
    <span class="text-[var(--brand-text-tertiary)]">{newProjectCopy.nameHint}</span>
  {/if}
</p>
