<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import { newProjectCopy } from '../content/templates-copy';
  import { randomProjectName } from '../logic/random-name';

  interface Props {
    name?: string | undefined;
  }

  let { name = $bindable('') }: Props = $props();
</script>

<label class="mt-4 block font-medium" for="project-name">Name</label>
<div class="join mt-1 w-full">
  <input id="project-name" class="input join-item w-full" maxlength="80" placeholder="Bakery website" required aria-describedby="project-name-hint" bind:value={name} />
  <button class="btn join-item btn-square" type="button" title={newProjectCopy.random} aria-label={newProjectCopy.random} onclick={() => { name = randomProjectName(); }}><Icon name="dice" class="size-5" /></button>
</div>
<p id="project-name-hint" class="mt-1 text-sm text-[var(--brand-text-tertiary)]">{newProjectCopy.nameHint}</p>
