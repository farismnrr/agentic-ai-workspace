<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import Notice from '../components/Notice.svelte';
  import { projects } from '../content/projects';
  import AppShell from '../shell/AppShell.svelte';
  import ProjectsTable from './ProjectsTable.svelte';

  let query = $state('');
  let added = $state(false);
  const visible = $derived(projects.filter((project) => project.name.toLowerCase().includes(query.trim().toLowerCase())));
</script>

<svelte:head><title>Projects · Masih Awam</title></svelte:head>

<AppShell crumbs={[{ label: 'Projects' }]} width="dense">
  <div class="flex flex-wrap items-center justify-between gap-4">
    <div>
      <h1 class="text-xl font-semibold tracking-tight">Projects</h1>
      <p class="mt-1 text-[var(--brand-text-secondary)]">Each project is one website or thing you're working on.</p>
    </div>
    <div class="flex items-center gap-3">
      <label class="input input-sm h-9 w-52 rounded-lg"><Icon name="search" class="size-4 opacity-60" /><input type="search" placeholder="Search projects" aria-label="Search projects" bind:value={query} /></label>
      <button class="btn btn-primary btn-sm rounded-lg" type="button" onclick={() => (added = true)}><Icon name="plus" class="size-4" /> New project</button>
    </div>
  </div>
  {#if added}<div class="mt-4"><Notice show /></div>{/if}

  {#if visible.length === 0}
    <p class="mt-10 text-[var(--brand-text-secondary)]">No project matches “{query}”.</p>
  {:else}
    <div class="mt-6"><ProjectsTable items={visible} /></div>
  {/if}
</AppShell>
