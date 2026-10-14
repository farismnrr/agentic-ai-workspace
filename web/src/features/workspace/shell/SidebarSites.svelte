<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import { sitesCopy } from '../content/sites-copy';
  import { runningSites, siteName } from '../logic/sites';
  import { stopSite, type Site } from '../logic/sites-api';
  import { projects, refresh, sites } from '../logic/workspace-store';

  const running = $derived(runningSites($sites));

  async function stop(site: Site): Promise<void> {
    try {
      await stopSite(site.projectId);
    } finally {
      await refresh();
    }
  }
</script>

{#if running.length > 0}
  <details class="group mt-3" open>
    <summary class="flex cursor-pointer list-none items-center gap-1 px-2 pb-1 text-xs text-[var(--brand-text-tertiary)]">
      {sitesCopy.sidebarTitle} <span class="transition-transform group-open:rotate-90" aria-hidden="true">›</span>
    </summary>
    <ul class="flex flex-col">
      {#each running as site (site.projectId)}
        {@const name = siteName(site, $projects)}
        <li class="flex items-center gap-1 rounded-lg px-2 py-1 hover:bg-base-100/70">
          <span class="status status-success shrink-0" aria-hidden="true"></span>
          <a class="min-w-0 flex-1 truncate py-0.5" href={site.url} target="_blank" rel="noopener noreferrer" title="Open {name}">{name}</a>
          <button class="btn btn-ghost btn-xs btn-square" type="button" aria-label="Stop {name}" title={sitesCopy.stop} onclick={() => void stop(site)}>
            <Icon name="square" class="size-3.5" />
          </button>
        </li>
      {/each}
    </ul>
  </details>
{/if}
