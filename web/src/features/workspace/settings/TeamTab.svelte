<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '../../../shared/api/http';
  import Icon from '../../../shared/ui/Icon.svelte';
  import { teamCopy } from '../content/team-copy';
  import { loadAgentTeam, newRole, resetAgentTeam, saveAgentTeam, type AgentTeam } from '../logic/agent-team-api';
  import SaveBar from './parts/SaveBar.svelte';
  import SettingsSection from './parts/SettingsSection.svelte';
  import TeamRole from './TeamRole.svelte';

  let team: AgentTeam | undefined = $state();
  let saving = $state(false);
  let message = $state('');
  let failed = $state(false);

  onMount(async () => {
    try {
      team = await loadAgentTeam();
    } catch {
      failed = true;
      message = teamCopy.loadFailed;
    }
  });

  function report(ok: boolean, text: string): void {
    failed = !ok;
    message = text;
  }

  async function save(): Promise<void> {
    if (team === undefined) return;
    saving = true;
    try {
      team = await saveAgentTeam(team);
      report(true, teamCopy.saved);
    } catch (error) {
      report(false, (error instanceof ApiError ? teamCopy.errors[error.code] : undefined) ?? teamCopy.failed);
    } finally {
      saving = false;
    }
  }

  async function reset(): Promise<void> {
    try {
      team = await resetAgentTeam();
      report(true, teamCopy.reset);
    } catch {
      report(false, teamCopy.failed);
    }
  }

  const add = (): void => {
    team?.roles.push({ ...newRole(), name: teamCopy.newRoleName });
  };
</script>

{#if team !== undefined}
  <SettingsSection title="Roles" description={teamCopy.rolesIntro}>
    {#snippet action()}
      <div class="flex gap-2">
        <button class="btn btn-ghost btn-xs" type="button" onclick={reset}>Reset to built-in</button>
        <button class="btn btn-xs" type="button" onclick={add}><Icon name="plus" class="size-3.5" />Add role</button>
      </div>
    {/snippet}
    {#each team.roles as role, index (role.key === '' ? `new-${String(index)}` : role.key)}
      <TeamRole bind:role={team.roles[index]} onremove={() => { team?.roles.splice(index, 1); }} />
    {/each}
  </SettingsSection>
  <SaveBar {saving} {message} {failed} onsave={save} />
{:else if failed}
  <p class="text-error" role="alert">{message}</p>
{:else}
  <span class="loading loading-spinner loading-sm" aria-label="Loading"></span>
{/if}
