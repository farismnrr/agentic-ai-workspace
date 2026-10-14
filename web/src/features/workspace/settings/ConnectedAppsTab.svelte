<script lang="ts">
  import EmptyState from '../../../shared/ui/EmptyState.svelte';
  import { onMount } from 'svelte';
  import { connectedAppsCopy } from '../content/connected';
  import { loadConnectedApps, revokeApp, type ConnectedApp } from '../logic/connectors-api';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import SettingsSection from './parts/SettingsSection.svelte';

  let apps: ConnectedApp[] = $state([]);
  let loading = $state(true);
  let failed = $state(false);
  let pending: ConnectedApp | undefined = $state();

  async function refresh(): Promise<void> {
    try {
      apps = await loadConnectedApps();
      failed = false;
    } catch {
      failed = true;
    } finally {
      loading = false;
    }
  }

  async function confirmRevoke(): Promise<void> {
    if (pending === undefined) return;
    try {
      await revokeApp(pending.id);
    } finally {
      pending = undefined;
      await refresh();
    }
  }

  const day = (iso: string): string => new Date(iso).toLocaleDateString(undefined, { dateStyle: 'medium' });

  onMount(refresh);
</script>

<SettingsSection title="Connected apps" description={connectedAppsCopy.intro}>
  {#if loading}
    <p class="py-3"><span class="loading loading-spinner loading-sm" aria-label="Loading"></span></p>
  {:else if failed}
    <p class="py-3 text-error" role="alert">{connectedAppsCopy.failed}</p>
  {:else if apps.length === 0}
    <EmptyState compact icon="link" title={connectedAppsCopy.empty.title} body={connectedAppsCopy.empty.body}>
      <a class="btn btn-outline btn-sm rounded-lg" href="/app/settings/spaces">Open Spaces & tools</a>
    </EmptyState>
  {:else}
    {#each apps as app (app.id)}
      <div class="flex flex-wrap items-start justify-between gap-3 py-3">
        <div class="min-w-0 flex-1">
          <p class="font-medium">{app.clientName} <span class="font-normal text-[var(--brand-text-tertiary)]">· {app.resourceName}</span></p>
          <ul class="mt-1 text-xs text-[var(--brand-text-secondary)]">
            {#each app.scopes as scope (scope.id)}<li>{scope.description}</li>{/each}
          </ul>
          <p class="mt-1 text-xs text-[var(--brand-text-tertiary)]">
            Connected {day(app.createdAt)} via {app.redirectHost} · {app.lastUsedAt === null ? 'Not used yet' : `Last used ${day(app.lastUsedAt)}`}
          </p>
        </div>
        <button class="btn btn-outline btn-error btn-sm" type="button" onclick={() => (pending = app)}>Remove access</button>
      </div>
    {/each}
  {/if}
</SettingsSection>
<ConfirmModal
  open={pending !== undefined}
  title="Remove access for {pending?.clientName ?? ''}?"
  body="It will stop working right away. You can connect it again later."
  confirmLabel="Remove access"
  cancelLabel="Keep access"
  oncancel={() => (pending = undefined)}
  onconfirm={confirmRevoke}
/>
