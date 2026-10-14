<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import { personaCopy } from '../content/models';
  import type { Rule } from '../logic/preferences-api';
  import SettingsSection from './parts/SettingsSection.svelte';

  interface Props {
    rules?: Rule[] | undefined;
  }

  let { rules = $bindable([]) }: Props = $props();
  let draft = $state('');

  function add(event: SubmitEvent): void {
    event.preventDefault();
    const text = draft.trim();
    if (text === '') return;
    rules.push({ id: '', text, on: true });
    draft = '';
  }
</script>

<SettingsSection title="Always on" description="Built-in safety rules. These can't be turned off.">
  {#each personaCopy.builtInRules as rule (rule)}
    <p class="flex items-center gap-2 py-2.5"><Icon name="lock" class="size-4 text-[var(--brand-text-tertiary)]" />{rule}</p>
  {/each}
</SettingsSection>

<SettingsSection title="Your rules" description="Applied to every task, every time your AI team runs.">
  {#each rules as rule, index (rule.id === '' ? `new-${String(index)}` : rule.id)}
    <div class="flex items-center gap-2 py-2.5">
      <span class="flex-1">{rule.text}</span>
      <input type="checkbox" class="toggle toggle-primary toggle-sm" aria-label="{rule.text}: on or off" bind:checked={rule.on} />
      <button class="btn btn-ghost btn-xs btn-square" type="button" aria-label="Delete rule: {rule.text}" onclick={() => (rules = rules.filter((_, at) => at !== index))}><Icon name="trash" class="size-4" /></button>
    </div>
  {/each}
  <form class="flex gap-2 py-3" onsubmit={add}>
    <label class="sr-only" for="new-rule">New rule</label>
    <input id="new-rule" class="input input-sm flex-1" placeholder="Add a rule in plain words…" bind:value={draft} />
    <button class="btn btn-sm" type="submit">Add</button>
  </form>
</SettingsSection>
