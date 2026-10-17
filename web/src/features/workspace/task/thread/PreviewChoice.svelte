<script lang="ts">
  import { readLabel } from '../../logic/question-form';
  import type { QuestionOption } from '../../logic/task-types';
  import { stylePreview } from '../../assets/style-previews';

  interface Props {
    option: QuestionOption;
    name: string;
    /** Pick several (checkboxes) instead of one (radio buttons). */
    multi: boolean;
    checked: boolean;
    onchange: (on: boolean) => void;
  }

  const { option, name, multi, checked, onchange }: Props = $props();
  const label = $derived(readLabel(option.label));
  const picture = $derived(option.preview === undefined ? undefined : stylePreview(option.preview));
</script>

<label class="card cursor-pointer overflow-hidden border border-base-300 bg-base-100 transition-shadow has-[:checked]:border-primary has-[:checked]:ring-2 has-[:checked]:ring-primary/40 has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-primary">
  {#if picture !== undefined}
    <img class="aspect-[8/5] w-full object-cover" src={picture} alt={`Preview of ${label.text}`} loading="lazy" />
  {/if}
  <span class="flex items-start gap-2 p-2.5">
    <input class="{multi ? 'checkbox' : 'radio'} {multi ? 'checkbox-sm' : 'radio-sm'} mt-0.5" type={multi ? 'checkbox' : 'radio'} {name} {checked} onchange={(event) => { onchange(event.currentTarget.checked); }} />
    <span class="flex min-w-0 flex-col text-sm">
      <span class="font-medium">{label.text}{#if label.recommended} <span class="badge badge-primary badge-sm ml-1 align-middle">Recommended</span>{/if}</span>
      {#if option.description !== undefined && option.description !== ''}<span class="text-xs text-[var(--brand-text-secondary)]">{option.description}</span>{/if}
    </span>
  </span>
</label>
