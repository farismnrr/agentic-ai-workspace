<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';
  import type { TaskEvent } from '../../logic/task-types';

  interface Props { event: Extract<TaskEvent, { kind: 'role' }> }
  const { event }: Props = $props();
  const next = $derived(event.work?.roleName ?? event.text);
  const previous = $derived(event.work?.from ?? 'No role selected');
</script>

<p class="flex min-w-0 items-center gap-2 text-sm" aria-label={`Role changed from ${previous} to ${next}`}>
  <Icon name="arrow-right" class="size-4 shrink-0 text-secondary" />
  <span class="truncate"><span class="text-[var(--brand-text-secondary)]">Role</span> · {previous} → <strong>{next}</strong></span>
</p>
