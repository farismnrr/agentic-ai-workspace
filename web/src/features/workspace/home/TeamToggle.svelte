<script lang="ts">
  import { onMount } from 'svelte';
  import { loadAgentTeam } from '../logic/agent-team-api';

  interface Props {
    team?: boolean | undefined;
    disabled?: boolean | undefined;
  }

  let { team = $bindable(false), disabled = false }: Props = $props();

  // Starts where the person left it in Settings, Agent team; a failed load leaves it off.
  onMount(async () => {
    try {
      team = (await loadAgentTeam()).useByDefault;
    } catch {
      team = false;
    }
  });
</script>

<label class="flex cursor-pointer items-center gap-1.5 rounded-full px-2 py-1 text-sm hover:bg-base-200 {team ? 'font-medium' : 'text-[var(--brand-text-secondary)]'}" title="Let a lead hand the work to your AI team">
  <input class="toggle toggle-xs toggle-primary" type="checkbox" bind:checked={team} {disabled} />Team
</label>
