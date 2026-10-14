<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import { statusMeta } from '../content/status';
  import type { TaskStatus } from '../content/task-types';
  import { iconTone } from './tone';

  interface Props {
    status: TaskStatus;
  }

  const { status }: Props = $props();
  const meta = $derived(statusMeta[status]);
</script>

<Icon name={meta.icon} class="size-[18px] shrink-0 {iconTone[meta.tone]} {status === 'working' ? 'motion-safe:animate-spin' : ''}" />
<span class="sr-only">{meta.label}</span>
