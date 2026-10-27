<script lang="ts">
  import { tick } from 'svelte';
  import EmptyState from '../../../shared/ui/EmptyState.svelte';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import { extensionsCopy } from '../content/extensions-copy';
  import { removeExtension, type Extension } from '../logic/extensions-api';
  import { extensions, loaded, refresh } from '../logic/workspace-store';
  import AppShell from '../shell/AppShell.svelte';
  import ExtensionCard from './ExtensionCard.svelte';
  import InstallGuide from './InstallGuide.svelte';

  interface Props {
    /** The extension whose install steps open first: the one the person came to install. */
    projectId?: string | undefined;
  }

  const { projectId }: Props = $props();
  let removing: Extension | undefined = $state();
  let error = $state('');

  const installing = $derived($extensions.find((item) => item.projectId === projectId));

  // Coming from the thread's Install button, the steps are what the person should see first.
  $effect(() => {
    if (installing !== undefined) void tick().then(() => document.getElementById(`install-${installing.projectId}`)?.scrollIntoView({ block: 'start' }));
  });

  async function confirmRemove(): Promise<void> {
    const target = removing;
    removing = undefined;
    if (target === undefined) return;
    try {
      await removeExtension(target.projectId);
    } catch {
      error = 'We could not remove that. Please try again.';
    }
    await refresh();
  }
</script>

<svelte:head><title>{extensionsCopy.title} · Masih Awam</title></svelte:head>

<AppShell crumbs={[{ label: 'Settings', href: '/app/settings' }, { label: extensionsCopy.title }]} width="dense">
  <h1 class="text-xl font-semibold tracking-tight">{extensionsCopy.title}</h1>
  <p class="mt-1 max-w-2xl text-[var(--brand-text-secondary)]">{extensionsCopy.intro}</p>
  {#if error !== ''}<p class="mt-3 text-error" role="alert">{error}</p>{/if}

  {#if installing !== undefined}
    <div class="mt-6"><InstallGuide extension={installing} /></div>
    <p class="mt-3 flex gap-2 text-sm text-[var(--brand-text-secondary)]">{extensionsCopy.trust}</p>
  {:else if projectId !== undefined && $loaded}
    <p class="mt-6 text-[var(--brand-text-secondary)]">That extension is not here any more.</p>
  {/if}

  {#if $extensions.length === 0 && $loaded}
    <EmptyState class="mt-6" bordered icon="puzzle" title={extensionsCopy.empty.title} body={extensionsCopy.empty.body}>
      <a class="btn btn-primary btn-sm rounded-lg" href="/app">{extensionsCopy.empty.action}</a>
    </EmptyState>
  {:else}
    <div class="mt-6 overflow-hidden card border border-[var(--brand-border)] bg-base-100">
      <ul class="divide-y divide-[var(--brand-border)]">
        {#each $extensions as extension (extension.projectId)}
          <ExtensionCard {extension} installing={installing?.projectId === extension.projectId} onremove={(target: Extension) => { removing = target; }} />
        {/each}
      </ul>
    </div>
  {/if}
</AppShell>

<ConfirmModal
  open={removing !== undefined}
  title={extensionsCopy.removeTitle}
  body={extensionsCopy.removeBody}
  confirmLabel={extensionsCopy.removeConfirm}
  cancelLabel={extensionsCopy.cancel}
  oncancel={() => (removing = undefined)}
  onconfirm={confirmRemove}
/>
