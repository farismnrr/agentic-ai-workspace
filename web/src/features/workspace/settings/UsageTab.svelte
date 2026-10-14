<script lang="ts">
  import EmptyState from '../../../shared/ui/EmptyState.svelte';
  import { onMount } from 'svelte';
  import { loadUsage, type UsageRow } from '../logic/tasks-api';
  import SettingRow from './parts/SettingRow.svelte';
  import SettingsSection from './parts/SettingsSection.svelte';

  let rows: UsageRow[] = $state([]);
  let since = $state('');
  let loading = $state(true);
  let failed = $state(false);

  const total = $derived(rows.reduce((sum, row) => sum + row.inputTokens + row.outputTokens, 0));
  const tasksTotal = $derived(rows.reduce((sum, row) => sum + row.tasks, 0));
  const tokens = (row: UsageRow): number => row.inputTokens + row.outputTokens;
  const format = (value: number): string => value.toLocaleString();

  onMount(async () => {
    try {
      const usage = await loadUsage();
      rows = usage.models;
      since = usage.since;
    } catch {
      failed = true;
    } finally {
      loading = false;
    }
  });
</script>

{#if loading}
  <span class="loading loading-spinner loading-sm" aria-label="Loading"></span>
{:else if failed}
  <p class="text-error" role="alert">We could not load your usage. Please try again in a moment.</p>
{:else}
  <SettingsSection title="This month" description="Since {new Date(since).toLocaleDateString(undefined, { dateStyle: 'long' })}. Providers bill you directly; these are token counts.">
    <div class="py-3">
      <p class="text-2xl font-semibold">{format(total)} <span class="text-sm font-normal text-[var(--brand-text-secondary)]">tokens across {tasksTotal} {tasksTotal === 1 ? 'task' : 'tasks'}</span></p>
    </div>
  </SettingsSection>

  <SettingsSection title="By model">
    {#each rows as row (`${row.providerId}:${row.model}`)}
      <SettingRow label={row.model} description="{row.tasks} {row.tasks === 1 ? 'task' : 'tasks'} · {format(tokens(row))} tokens">
        <progress class="progress progress-secondary h-1.5 w-40" value={tokens(row)} max={Math.max(total, 1)} aria-label="{row.model} share of tokens"></progress>
      </SettingRow>
    {:else}
      <EmptyState compact icon="bar-chart" title="Nothing used yet this month" body="Usage by model appears here once your AI team has done some work." />
    {/each}
  </SettingsSection>
{/if}
