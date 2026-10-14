<script lang="ts">
  import { findProject, runningTasks } from '../logic/derive';
  import { projects, tasks } from '../logic/workspace-store';

  const running = $derived(runningTasks($tasks));
</script>

{#if running.length > 0}
  <section aria-labelledby="running">
    <h2 id="running" class="mb-1.5 text-xs font-medium text-[var(--brand-text-secondary)]">Working on it · {running.length}</h2>
    <ul class="divide-y divide-[var(--brand-border)] rounded-xl border border-[var(--brand-border)] bg-base-100">
      {#each running as task (task.id)}
        {@const done = task.plan.filter((step) => step.state === 'done').length}
        {@const current = task.plan.find((step) => step.state === 'active')}
        <li>
          <a class="block px-3 py-2.5 hover:bg-base-200/60" href="/app/projects/{task.projectId}/tasks/{task.id}">
            <span class="flex items-baseline justify-between gap-3"><span class="font-medium">{task.title}</span><span class="text-xs text-[var(--brand-text-tertiary)]">{findProject($projects, task.projectId)?.name}</span></span>
            {#if task.plan.length > 0}
              <progress class="progress progress-primary mt-2 h-1" value={done} max={task.plan.length} aria-label="Progress"></progress>
              <span class="mt-1.5 block text-xs text-[var(--brand-text-secondary)]">{current?.title ?? 'Working'} · step {Math.min(done + 1, task.plan.length)} of {task.plan.length}</span>
            {:else}
              <span class="mt-1.5 block text-xs text-[var(--brand-text-secondary)]">Making a plan…</span>
            {/if}
          </a>
        </li>
      {/each}
    </ul>
  </section>
{/if}
