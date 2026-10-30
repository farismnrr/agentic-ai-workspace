<script lang="ts">
  import DocText from '../../components/DocText.svelte';
  import EmptyState from '../../../../shared/ui/EmptyState.svelte';
  import Icon from '../../../../shared/ui/Icon.svelte';
  import type { TaskEvent, WorkSummary } from '../../logic/task-types';
  import { workRoleSegments, type WorkRoleState } from '../../logic/work-roles';

  interface Props { events: TaskEvent[]; actor: string; work?: WorkSummary | undefined }
  const { events, actor, work }: Props = $props();
  const roles = $derived(work === undefined ? [] : workRoleSegments(events, work));
  const states: Record<WorkRoleState, { label: string; icon: 'check-circle' | 'activity' | 'clock'; tone: string }> = {
    done: { label: 'Done', icon: 'check-circle', tone: 'badge-success' },
    active: { label: 'In progress', icon: 'activity', tone: 'badge-info' },
    pending: { label: 'Pending', icon: 'clock', tone: 'badge-ghost' },
    waiting: { label: 'Awaiting approval', icon: 'clock', tone: 'badge-warning' },
  };
</script>

<section class="flex flex-col gap-3" aria-label="External assistant work">
  <header class="border-b border-[var(--brand-border)] pb-3">
    <h2 class="font-semibold">{actor} — External Assistant</h2>
    <p class="mt-1 text-xs text-[var(--brand-text-secondary)]">Role assignments, objectives and outcomes.</p>
  </header>
  {#if roles.length === 0}
    <EmptyState compact icon="activity" title="No structured role updates" body="This older external task has no role, step or result details to show." />
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
          {#if role.reports.length > 0}
            <div class="mt-2 text-sm text-[var(--brand-text-secondary)]">
              <DocText text={role.reports.at(-1) ?? ''} />
            </div>
          {/if}
        </li>
      {/each}
    </ol>
  {/if}
</section>
