<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import { statusMeta } from '../content/status';
  import type { TaskStatus } from '../content/task-types';
  import { badgeTone } from './tone';

  interface Props {
    status: TaskStatus;
  }

  const { status }: Props = $props();
  const meta = $derived(statusMeta[status]);
</script>

<span class="badge gap-1.5 whitespace-nowrap {badgeTone[meta.tone]}">
  <Icon name={meta.icon} class="size-3.5 {status === 'working' ? 'motion-safe:animate-spin' : ''}" />{meta.label}
</span>
