<script lang="ts">
  import { usageByModel } from '../content/settings';
  import { plan } from '../content/user';
  import SettingRow from './parts/SettingRow.svelte';
  import SettingsSection from './parts/SettingsSection.svelte';
</script>

<SettingsSection title="This month" description="{plan.name}. Resets on the 1st. Tasks you stop early don't count.">
  <div class="py-3">
    <p class="text-2xl font-semibold">{plan.used} <span class="text-sm font-normal text-[var(--brand-text-secondary)]">of {plan.limit} tasks</span></p>
    <progress class="progress progress-primary mt-2 h-1.5" value={plan.used} max={plan.limit} aria-label="Tasks used this month"></progress>
  </div>
</SettingsSection>

<SettingsSection title="By model">
  {#each usageByModel as row (row.name)}
    <SettingRow label={row.name} description="{String(row.tasks)} tasks"><progress class="progress progress-secondary h-1.5 w-40" value={row.share} max="100" aria-label="{row.name} share of tasks"></progress></SettingRow>
  {/each}
</SettingsSection>
