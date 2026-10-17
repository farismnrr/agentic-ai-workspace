<script lang="ts">
  import type { Draft } from '../../logic/question-form';
  import type { Question } from '../../logic/task-types';
  import PreviewChoice from './PreviewChoice.svelte';

  interface Props {
    question: Question;
    /** Makes the radio group and the checkboxes of this question their own. */
    name: string;
    draft: Draft;
    onchange: (draft: Draft) => void;
    /** The only question of its form: the card already shows its text. */
    single?: boolean;
  }

  const { question, name, draft, onchange, single = false }: Props = $props();

  const withPictures = $derived((question.options ?? []).some((option) => option.preview !== undefined));

  /** A picture card was ticked or cleared: several can be ticked in a multi question, one in a single question. */
  function choose(label: string, on: boolean): void {
    if (question.type === 'multi') toggle(label, on);
    else onchange({ ...draft, picked: [label] });
  }

  function toggle(label: string, on: boolean): void {
    onchange({ ...draft, picked: on ? [...draft.picked, label] : draft.picked.filter((picked) => picked !== label) });
  }
</script>

<fieldset class="fieldset gap-1.5">
  <legend class="fieldset-legend flex items-center gap-2 p-0 text-sm font-medium" class:sr-only={single}>
    {#if question.header !== undefined && question.header !== ''}<span class="badge badge-outline badge-sm">{question.header}</span>{/if}
    {question.text}
  </legend>
  {#if question.type === 'multi'}<p class="text-xs text-[var(--brand-text-secondary)]">Pick all that apply.</p>{/if}
  {#if withPictures}
    <div class="grid grid-cols-1 gap-3 min-[480px]:grid-cols-2">
      {#each question.options ?? [] as option (option.label)}
        <PreviewChoice {option} {name} multi={question.type === 'multi'} checked={draft.picked.includes(option.label)} onchange={(on: boolean) => { choose(option.label, on); }} />
      {/each}
    </div>
  {:else}
    {#each question.options ?? [] as option (option.label)}
      <label class="flex cursor-pointer items-start gap-2.5 rounded-lg border border-base-300 p-2.5 has-[:checked]:border-primary">
        {#if question.type === 'multi'}
          <input class="checkbox checkbox-sm mt-0.5" type="checkbox" {name} checked={draft.picked.includes(option.label)} onchange={(event) => { toggle(option.label, event.currentTarget.checked); }} />
        {:else}
          <input class="radio radio-sm mt-0.5" type="radio" {name} checked={draft.picked.includes(option.label)} onchange={() => { onchange({ ...draft, picked: [option.label] }); }} />
        {/if}
        <span class="flex flex-col text-sm">
          <span class="font-medium">{option.label}</span>
          {#if option.description !== undefined && option.description !== ''}<span class="text-xs text-[var(--brand-text-secondary)]">{option.description}</span>{/if}
        </span>
      </label>
    {/each}
  {/if}
  <input class="input input-sm h-10 w-full" type="text" placeholder={question.type === 'text' ? 'Your answer' : 'Other: type your own answer'} aria-label={`Your own answer to: ${question.text}`} value={draft.other} oninput={(event) => { onchange({ ...draft, other: event.currentTarget.value }); }} />
</fieldset>
