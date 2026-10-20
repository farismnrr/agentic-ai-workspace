<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import ModelPicker from '../home/ModelPicker.svelte';
  import { accessSummary, teamCopy } from '../content/team-copy';
  import { applyPick, modelPick, newRole, type TeamRole } from '../logic/agent-team-api';

  interface Props {
    role?: TeamRole | undefined;
    onremove: () => void;
  }

  let { role = $bindable(newRole()), onremove }: Props = $props();
  const id = $derived(`role-${role.key === '' ? 'new' : role.key}`);
  const groups = ['write', 'exec', 'ask', 'web', 'images'] as const;

  function allow(group: string, on: boolean): void {
    role.access = on ? [...role.access.filter((item) => item !== group), group] : role.access.filter((item) => item !== group);
  }
</script>

<details class="group py-2">
  <summary class="flex cursor-pointer list-none items-center gap-2 py-1">
    <Icon name="chevron-right" class="size-4 shrink-0 transition-transform group-open:rotate-90" />
    <span class="font-medium">{role.name === '' ? teamCopy.newRoleName : role.name}</span>
    {#if role.lead}<span class="badge badge-sm badge-primary badge-soft">Lead</span>{:else if !role.enabled}<span class="badge badge-sm badge-ghost">Off</span>{/if}
    <span class="hidden min-w-0 flex-1 truncate text-xs text-[var(--brand-text-tertiary)] sm:block">{role.lead ? teamCopy.leadNote : role.purpose}</span>
  </summary>
  <div class="mt-3 flex flex-col gap-3 pl-6">
    <div class="grid gap-3 sm:grid-cols-2">
      <div class="flex flex-col gap-1">
        <label class="text-sm font-medium" for="{id}-name">Name</label>
        <input id="{id}-name" class="input input-sm w-full" maxlength="40" placeholder="Data analyst" bind:value={role.name} />
      </div>
      <div class="flex flex-col gap-1">
        <label class="text-sm font-medium" for="{id}-purpose">What it is for</label>
        <input id="{id}-purpose" class="input input-sm w-full" maxlength="200" placeholder="Reads the numbers and explains them" bind:value={role.purpose} />
      </div>
    </div>
    <div class="flex flex-col gap-1">
      <label class="text-sm font-medium" for="{id}-prompt">Personality and way of working</label>
      <textarea id="{id}-prompt" class="textarea w-full" rows="4" maxlength="2000" bind:value={role.prompt}></textarea>
    </div>
    <div class="flex flex-wrap items-center gap-2">
      <span class="text-sm font-medium">Model</span>
      <ModelPicker bind:model={() => modelPick(role), (value: string) => { applyPick(role, value); }} bind:effort={role.effort} />
    </div>
    {#if !role.lead && role.preset}
      <p class="text-sm text-[var(--brand-text-secondary)]">{accessSummary(role.access)} <span class="text-xs text-[var(--brand-text-tertiary)]">{teamCopy.builtInNote}</span></p>
    {/if}
    {#if !role.lead && !role.preset}
      <fieldset class="flex flex-col gap-1.5">
        <legend class="mb-1 text-sm font-medium">What it may do</legend>
        <label class="flex items-center gap-2 text-sm text-[var(--brand-text-secondary)]">
          <input class="checkbox checkbox-sm" type="checkbox" checked disabled />{teamCopy.access['read']?.title}
        </label>
        {#each groups as group (group)}
          <label class="flex items-center gap-2 text-sm">
            <input class="checkbox checkbox-sm checkbox-primary" type="checkbox" checked={role.access.includes(group)} onchange={(event) => { allow(group, event.currentTarget.checked); }} />
            <span>{teamCopy.access[group]?.title} <span class="text-xs text-[var(--brand-text-tertiary)]">{teamCopy.access[group]?.note}</span></span>
          </label>
        {/each}
      </fieldset>
    {/if}
    {#if !role.lead}
      <div class="flex items-center justify-between gap-3">
        <label class="flex items-center gap-2 text-sm font-medium"><input class="toggle toggle-sm toggle-primary" type="checkbox" bind:checked={role.enabled} />On the team</label>
        {#if !role.preset}<button class="btn btn-ghost btn-sm text-error" type="button" onclick={onremove}><Icon name="trash" class="size-4" />Remove</button>{/if}
      </div>
    {/if}
  </div>
</details>
