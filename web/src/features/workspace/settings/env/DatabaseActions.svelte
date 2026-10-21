<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';

  interface Props { busy: boolean; testing: boolean; configured: boolean; ontest: () => void; onsave: () => void; ondelete: () => void }
  const { busy, testing, configured, ontest, onsave, ondelete }: Props = $props();
</script>

<div class="flex flex-wrap gap-2">
  <button class="btn btn-outline btn-sm min-h-11" type="button" onclick={ontest} disabled={busy} aria-busy={busy}>
    {#if testing}<span class="loading loading-spinner loading-xs" aria-hidden="true"></span>{:else}<Icon name="activity" class="size-4" />{/if}
    Test connection
  </button>
  <button class="btn btn-primary btn-sm rounded-lg min-h-11" type="button" onclick={onsave} disabled={busy}>Save</button>
  {#if configured}<button class="btn btn-ghost btn-sm min-h-11 text-error" type="button" onclick={ondelete} disabled={busy}>Delete</button>{/if}
</div>
