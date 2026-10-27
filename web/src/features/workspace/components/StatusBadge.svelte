<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import { statusMeta } from '../content/status';
  import type { TaskStatus } from '../logic/task-types';
  import { badgeTone } from './tone';

  interface Props {
    status: TaskStatus;
    label?: string;
  }

  const { status, label }: Props = $props();
  const meta = $derived(statusMeta[status]);
  const shownLabel = $derived(label ?? meta.label);
</script>

<span class="badge gap-1.5 whitespace-nowrap {badgeTone[meta.tone]}">
  <Icon name={meta.icon} class="size-3.5 {status === 'working' ? 'motion-safe:animate-spin' : ''}" />{shownLabel}
</span>
