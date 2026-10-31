<script lang="ts">
  import { approvalsCopy } from '../content/connected';
  import { setSkipApprovals, type ConnectorSpace } from '../logic/connectors-api';

  interface Props {
    space: ConnectorSpace;
  }

  const { space }: Props = $props();
  let override: boolean | undefined = $state();
  let failed = $state(false);
  const on = $derived(override ?? space.skipApprovals);

  async function flip(skip: boolean): Promise<void> {
    override = skip;
    failed = false;
    try {
      await setSkipApprovals(space.slug, skip);
    } catch {
      override = !skip;
      failed = true;
    }
  }
</script>

<div class="flex flex-wrap items-center gap-3 border-b border-[var(--brand-border)] px-4 py-3">
  <div class="min-w-44 flex-1 leading-tight">
    <p class="font-medium">{approvalsCopy.title}</p>
    <p class="text-xs text-[var(--brand-text-secondary)]">{approvalsCopy.body}</p>
    {#if failed}<p class="text-xs text-error" role="alert">{approvalsCopy.failed}</p>{/if}
  </div>
  <input class="toggle toggle-sm toggle-warning" type="checkbox" aria-label={approvalsCopy.title} checked={on} onchange={(event) => flip(event.currentTarget.checked)} />
</div>
