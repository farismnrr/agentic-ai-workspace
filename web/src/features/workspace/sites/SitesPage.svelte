<script lang="ts">
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import { sitesCopy } from '../content/sites-copy';
  import { describeSiteError } from '../logic/site-errors';
  import { orderedSites } from '../logic/sites';
  import { setVisibility, stopSite, type Site } from '../logic/sites-api';
  import { projects, refresh, sites } from '../logic/workspace-store';
  import AppShell from '../shell/AppShell.svelte';
  import SiteRow from './SiteRow.svelte';

  let publishing: Site | undefined = $state();
  let stopping: Site | undefined = $state();
  let busy = $state(false);
  let error = $state('');

  async function run(action: () => Promise<undefined>): Promise<void> {
    busy = true;
    error = '';
    try {
      await action();
    } catch (failure) {
      error = describeSiteError(failure);
    } finally {
      busy = false;
      await refresh();
    }
  }

  function confirmPublic(): void {
    const target = publishing;
    publishing = undefined;
    if (target !== undefined) void run(() => setVisibility(target.projectId, 'public'));
  }

  function confirmStop(): void {
    const target = stopping;
    stopping = undefined;
    if (target !== undefined) void run(() => stopSite(target.projectId));
  }
</script>

<svelte:head><title>{sitesCopy.title} · Masih Awam</title></svelte:head>

<AppShell crumbs={[{ label: sitesCopy.title }]} width="dense">
  <h1 class="text-xl font-semibold tracking-tight">{sitesCopy.title}</h1>
  <p class="mt-1 max-w-2xl text-[var(--brand-text-secondary)]">{sitesCopy.intro}</p>
  {#if error !== ''}<p class="mt-3 text-error" role="alert">{error}</p>{/if}

  {#if $sites.length === 0}
    <p class="mt-10 text-[var(--brand-text-secondary)]">{sitesCopy.none}</p>
  {:else}
    <div class="mt-6 overflow-hidden card border border-[var(--brand-border)] bg-base-100">
      <ul class="divide-y divide-[var(--brand-border)]">
        {#each orderedSites($sites) as site (site.projectId)}
          <SiteRow
            {site}
            projects={$projects}
            {busy}
            onpublic={(target: Site) => (publishing = target)}
            onprivate={(target: Site) => void run(() => setVisibility(target.projectId, 'private'))}
            onstop={(target: Site) => (stopping = target)}
          />
        {/each}
      </ul>
    </div>
  {/if}
</AppShell>

<ConfirmModal
  open={publishing !== undefined}
  title={sitesCopy.publicTitle}
  body={sitesCopy.publicBody}
  confirmLabel={sitesCopy.publicConfirm}
  cancelLabel={sitesCopy.cancel}
  oncancel={() => (publishing = undefined)}
  onconfirm={confirmPublic}
/>
<ConfirmModal
  open={stopping !== undefined}
  title={sitesCopy.stopTitle}
  body={sitesCopy.stopBody}
  confirmLabel={sitesCopy.stopConfirm}
  cancelLabel={sitesCopy.cancel}
  oncancel={() => (stopping = undefined)}
  onconfirm={confirmStop}
/>
