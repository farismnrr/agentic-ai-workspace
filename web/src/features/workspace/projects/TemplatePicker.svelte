<script lang="ts">
  import { emptyStart, startCopy, templateStarts } from '../content/templates-copy';

  interface Props {
    value?: string | undefined;
  }

  let { value = $bindable('') }: Props = $props();
  const starts = [emptyStart, ...templateStarts];
</script>

<fieldset class="mt-4">
  <legend class="font-medium">{startCopy.label}</legend>
  <div class="mt-2 grid max-h-56 grid-cols-2 gap-2 overflow-y-auto p-1">
    {#each starts as start (start.id)}
      <label class="card card-border cursor-pointer p-3 text-sm has-[:checked]:border-primary has-[:checked]:bg-base-200 has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-primary">
        <input type="radio" name="project-start" class="sr-only" value={start.id} bind:group={value} />
        <span class="font-medium">{start.title}</span>
        <span class="text-[var(--brand-text-secondary)]">{start.body}</span>
      </label>
    {/each}
  </div>
  {#if value !== ''}<p class="mt-2 text-sm text-[var(--brand-text-secondary)]">{startCopy.note}</p>{/if}
</fieldset>
