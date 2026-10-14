<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import { sitesCopy } from '../content/sites-copy';
  import { absoluteUrl, siteName } from '../logic/sites';
  import type { Site } from '../logic/sites-api';
  import type { Project } from '../logic/task-types';

  interface Props {
    site: Site;
    projects: Project[];
    busy: boolean;
    onpublic: (site: Site) => void;
    onprivate: (site: Site) => void;
    onstop: (site: Site) => void;
  }

  const { site, projects, busy, onpublic, onprivate, onstop }: Props = $props();
  const running = $derived(site.status === 'running');
  const isPublic = $derived(site.visibility === 'public');
  let copied = $state(false);

  async function copy(): Promise<void> {
    await navigator.clipboard.writeText(absoluteUrl(site));
    copied = true;
    setTimeout(() => (copied = false), 1500);
  }
</script>

<li class="flex flex-wrap items-center gap-3 px-4 py-3 {running ? '' : 'opacity-60'}">
  <div class="min-w-0 flex-1">
    <p class="flex items-center gap-2 font-medium">
      <span class="status {running ? 'status-success' : 'status-neutral'}" aria-hidden="true"></span>
      <span class="truncate">{siteName(site, projects)}</span>
      <span class="badge badge-sm {running ? 'badge-success badge-soft' : 'badge-ghost'}">{running ? sitesCopy.running : sitesCopy.stopped}</span>
    </p>
    <p class="mt-1 text-xs text-[var(--brand-text-secondary)]">{isPublic ? sitesCopy.publicLabel : sitesCopy.privateLabel}</p>
  </div>
  <label class="flex items-center gap-2 text-sm">
    <input
      type="checkbox"
      class="toggle toggle-sm toggle-primary"
      checked={isPublic}
      disabled={busy}
      aria-label="Public: {siteName(site, projects)}"
      onchange={() => { if (isPublic) onprivate(site); else onpublic(site); }}
    />
    Public
  </label>
  {#if running}
    <a class="btn btn-primary btn-sm rounded-lg" href={site.url} target="_blank" rel="noopener noreferrer">{sitesCopy.open} <Icon name="external-link" class="size-4" /></a>
    <button class="btn btn-ghost btn-sm" type="button" onclick={() => void copy()}>{copied ? sitesCopy.copied : sitesCopy.copy}</button>
    <button class="btn btn-outline btn-sm rounded-lg" type="button" disabled={busy} onclick={() => { onstop(site); }}><Icon name="square" class="size-4" /> {sitesCopy.stop}</button>
  {/if}
</li>
