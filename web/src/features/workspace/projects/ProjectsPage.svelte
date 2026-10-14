<script lang="ts">
  import EmptyState from '../../../shared/ui/EmptyState.svelte';
  import { navigate } from '../../../app/router';
  import Icon from '../../../shared/ui/Icon.svelte';
  import { projectsCopy } from '../content/projects-copy';
  import type { Project } from '../logic/task-types';
  import { projects, refresh } from '../logic/workspace-store';
  import AppShell from '../shell/AppShell.svelte';
  import NewProjectModal from './NewProjectModal.svelte';
  import ProjectsTable from './ProjectsTable.svelte';

  let query = $state('');
  let adding = $state(false);
  const visible = $derived($projects.filter((project) => project.name.toLowerCase().includes(query.trim().toLowerCase())));

  async function created(project: Project): Promise<void> {
    await refresh();
    navigate(`/app/projects/${project.id}`);
  }
</script>

<svelte:head><title>Projects · Masih Awam</title></svelte:head>

<AppShell crumbs={[{ label: 'Projects' }]} width="dense">
  <div class="flex flex-wrap items-center justify-between gap-4">
    <div>
      <h1 class="text-xl font-semibold tracking-tight">Projects</h1>
      <p class="mt-1 text-[var(--brand-text-secondary)]">{projectsCopy.intro}</p>
    </div>
    <div class="flex items-center gap-3">
      <label class="input input-sm h-9 w-52 rounded-lg"><Icon name="search" class="size-4 opacity-60" /><input type="search" placeholder="Search projects" aria-label="Search projects" bind:value={query} /></label>
      <button class="btn btn-primary btn-sm rounded-lg" type="button" onclick={() => (adding = true)}><Icon name="plus" class="size-4" /> New project</button>
    </div>
  </div>

  {#if $projects.length === 0}
    <EmptyState class="mt-6" bordered icon="folder" title={projectsCopy.empty.title} body={projectsCopy.empty.body} />
  {:else if visible.length === 0}
    <EmptyState class="mt-6" bordered icon="search" title="No project matches “{query}”" body="Try a different name, or clear the search." />
  {:else}
    <div class="mt-6"><ProjectsTable items={visible} /></div>
  {/if}
</AppShell>
<NewProjectModal bind:show={adding} oncreated={created} />
