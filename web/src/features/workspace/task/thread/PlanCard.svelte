<script lang="ts">
  import DocText from '../../components/DocText.svelte';
  import Icon from '../../../../shared/ui/Icon.svelte';
  import type { PlanState } from '../../logic/task-types';

  interface Props {
    text: string;
    /** One line per step, as "[state] title". */
    steps: string;
  }

  const { text, steps }: Props = $props();

  const marks: Record<PlanState, string> = { done: 'Done', active: 'Now', pending: 'Next', failed: 'Stuck' };
  const rows = $derived(
    steps
      .split('\n')
      .map((line) => /^\[(\w+)\] (.*)$/.exec(line))
      .filter((match): match is RegExpExecArray => match !== null)
      .map((match) => ({ state: (match[1] ?? 'pending') as PlanState, title: match[2] ?? '' })),
  );
</script>

<details class="group rounded-lg border border-[var(--brand-border)] bg-base-100 px-3 py-2">
  <summary class="flex cursor-pointer list-none items-center gap-2">
    <Icon name="check-circle" class="size-4 shrink-0 text-primary" />
    <span class="min-w-0 flex-1 truncate font-medium">{text}</span>
    <Icon name="chevron-down" class="size-3.5 shrink-0 transition-transform group-open:rotate-180" />
  </summary>
  <ul class="mt-2 flex flex-col gap-1 text-sm">
    {#each rows as row, index (index)}
      <li class="flex gap-2">
        <span class="badge badge-sm shrink-0 {row.state === 'failed' ? 'badge-error' : row.state === 'done' ? 'badge-primary' : 'badge-outline'}">{marks[row.state]}</span>
        <span class="min-w-0"><DocText text={row.title} /></span>
      </li>
    {/each}
  </ul>
</details>
