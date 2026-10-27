<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import { statusMeta } from '../content/status';
  import type { TaskStatus } from '../logic/task-types';
  import { iconTone } from './tone';

  interface Props {
    status: TaskStatus;
    label?: string;
  }

  const { status, label }: Props = $props();
  const meta = $derived(statusMeta[status]);
  const accessibleLabel = $derived(label ?? meta.label);
</script>

<Icon name={meta.icon} class="size-[18px] shrink-0 {iconTone[meta.tone]} {status === 'working' ? 'motion-safe:animate-spin' : ''}" />
<span class="sr-only">{accessibleLabel}</span>
