<script lang="ts">
  import EmptyState from '../../../../shared/ui/EmptyState.svelte';
  import Icon from '../../../../shared/ui/Icon.svelte';
  import type { TaskEvent } from '../../logic/task-types';
  import { teamJobs, type JobState } from '../../logic/team';

  interface Props {
    events: TaskEvent[];
    /** The task is waiting for the person, so an open question or approval is theirs to answer. */
    waiting: boolean;
  }

  const { events, waiting }: Props = $props();
  const jobs = $derived(teamJobs(events));
  const labels: Record<JobState, string> = { working: 'Working', waiting: 'Waiting for you', done: 'Done', failed: "Couldn't finish" };
  const tones: Record<JobState, string> = { working: 'badge-info', waiting: 'badge-warning', done: 'badge-success', failed: 'badge-error' };
  const stateOf = (state: JobState): JobState => (state === 'waiting' && !waiting ? 'working' : state);
  const count = (n: number, one: string, many: string): string => `${String(n)} ${n === 1 ? one : many}`;
</script>

{#if jobs.length === 0}
  <EmptyState compact bordered icon="user" title="No one has been called in yet" body="When the lead hands a job to a teammate, who is doing what shows up here." />
{:else}
  <ul class="flex flex-col gap-2">
    {#each jobs as job (job.id)}
      {@const state = stateOf(job.state)}
      <li class="rounded-lg border border-[var(--brand-border)] bg-base-100 p-3 text-sm">
        <p class="flex flex-wrap items-center gap-2">
          <span class="font-medium">{job.agent}</span>
          <span class="badge badge-soft badge-sm {tones[state]}">{labels[state]}</span>
          {#if job.model !== ''}<span class="text-xs text-[var(--brand-text-tertiary)]">{job.model}</span>{/if}
        </p>
        <p class="mt-1.5 text-[var(--brand-text-secondary)]">{job.brief}</p>
        <p class="mt-2 flex items-center gap-1.5 text-xs text-[var(--brand-text-tertiary)]">
          <Icon name="activity" class="size-3.5" />{count(job.steps, 'step', 'steps')} · {count(job.files.length, 'file changed', 'files changed')} · {count(job.checks, 'check', 'checks')}
        </p>
        {#if job.report !== ''}
          <details class="mt-2">
            <summary class="cursor-pointer text-xs font-medium text-primary">Report</summary>
            <p class="mt-1 whitespace-pre-wrap wrap-anywhere rounded-md bg-base-200 px-3 py-2">{job.report}</p>
          </details>
        {/if}
      </li>
    {/each}
  </ul>
{/if}
