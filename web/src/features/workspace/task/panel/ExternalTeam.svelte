<script lang="ts">
  import DocText from '../../components/DocText.svelte';
  import EmptyState from '../../../../shared/ui/EmptyState.svelte';
  import Icon from '../../../../shared/ui/Icon.svelte';
  import type { TaskEvent, WorkSummary } from '../../logic/task-types';
  import { workRoleSegments, type WorkRoleState } from '../../logic/work-roles';
  import EvidenceBadge from '../thread/EvidenceBadge.svelte';

  interface Props { events: TaskEvent[]; actor: string; work?: WorkSummary | undefined }
  const { events, actor, work }: Props = $props();
  const roles = $derived(work === undefined ? [] : workRoleSegments(events, work));
  const states: Record<WorkRoleState, { label: string; icon: 'check-circle' | 'activity' | 'clock'; tone: string }> = {
    done: { label: 'Done', icon: 'check-circle', tone: 'badge-success' },
    active: { label: 'In progress', icon: 'activity', tone: 'badge-info' },
    pending: { label: 'Pending', icon: 'clock', tone: 'badge-ghost' },
  };
</script>

<section class="flex flex-col gap-3" aria-label="External assistant work">
  <header class="border-b border-[var(--brand-border)] pb-3">
    <h2 class="font-semibold">{actor} — External Assistant</h2>
    <p class="mt-1 text-xs text-[var(--brand-text-secondary)]">Roles are labels one assistant chose; permissions never change.</p>
  </header>
  {#if roles.length === 0}
    <EmptyState compact bordered icon="activity" title="No structured role updates" body="This older external task has no role, step or result details to show." />
  {:else}
    <ol class="flex flex-col gap-2">
      {#each roles as role (role.id)}
        {@const status = states[role.state]}
        <li class="min-w-0 rounded-lg border border-[var(--brand-border)] bg-base-100 p-3">
          <div class="flex min-w-0 flex-wrap items-center gap-2">
            <h3 class="min-w-0 flex-1 truncate text-sm font-medium">{role.name}</h3>
            <span class="badge badge-soft badge-sm gap-1 {status.tone}"><Icon name={status.icon} class="size-3.5" />{status.label}</span>
          </div>
          {#if role.brief !== ''}<DocText class="mt-2 text-sm text-[var(--brand-text-secondary)]" text={role.brief} />{/if}
          {#if role.state !== 'pending'}
            <p class="mt-2 flex flex-wrap gap-x-3 gap-y-1 text-xs text-[var(--brand-text-secondary)]">
              <span>{role.steps.length} {role.steps.length === 1 ? 'step' : 'steps'}</span>
              <span>{role.files.length} {role.files.length === 1 ? 'file seen' : 'files seen'}</span>
              <span>{role.runs.length} {role.runs.length === 1 ? 'check run' : 'checks run'}</span>
              <span>{role.reports.length} {role.reports.length === 1 ? 'report' : 'reports'}</span>
            </p>
            {#if role.steps.length > 0}
              <ul class="mt-2 flex flex-col gap-1 text-sm">
                {#each role.steps as step (step.id)}
                  <li class="min-w-0 wrap-anywhere"><Icon name="pencil" class="mr-1 inline size-3.5 text-secondary" />{step.text}<EvidenceBadge work={step.work} exitCode={step.run?.exitCode} /></li>
                {/each}
              </ul>
            {/if}
            {#if role.files.length > 0}<p class="mt-2 flex flex-wrap gap-1">{#each role.files as file, index (index)}<code class="rounded bg-base-200 px-1.5 py-0.5 text-xs">{file}</code>{/each}</p>{/if}
            {#if role.reports.length > 0}
              <div class="mt-2 flex flex-col gap-1">{#each role.reports as report, index (index)}<div class="min-w-0 wrap-anywhere text-sm"><DocText text={report} /> <EvidenceBadge work={{ id: role.id, evidence: 'claimed' }} /></div>{/each}</div>
            {/if}
            {#if role.runs.length > 0}<p class="mt-2 flex flex-wrap gap-1">{#each role.runs as code, index (index)}<span class="badge badge-outline badge-xs">Check exit {code}</span>{/each}</p>{/if}
          {/if}
        </li>
      {/each}
    </ol>
  {/if}
</section>
