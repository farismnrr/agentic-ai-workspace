<script lang="ts">
  import { navigate } from '../../../app/router';
  import Icon from '../../../shared/ui/Icon.svelte';
  import ProjectTile from '../components/ProjectTile.svelte';
  import { findProject } from '../logic/derive';
  import { chosenProject, projects } from '../logic/workspace-store';
  import AppShell from '../shell/AppShell.svelte';
  import AboutTab from './AboutTab.svelte';
  import FileBrowser from '../files/FileBrowser.svelte';
  import ChecksTab from './ChecksTab.svelte';
  import TasksTab from './TasksTab.svelte';

  interface Props {
    projectId: string;
  }

  type Tab = 'tasks' | 'files' | 'checks' | 'about';

  const { projectId }: Props = $props();
  const project = $derived(findProject($projects, projectId));
  const tabs: { id: Tab; label: string }[] = [
    { id: 'tasks', label: 'Tasks' },
    { id: 'files', label: 'Files' },
    { id: 'checks', label: 'Checks' },
    { id: 'about', label: 'About' },
  ];
  let tab = $state<Tab>('tasks');

  function newTask(): void {
    chosenProject.set(projectId);
    navigate('/app');
  }
</script>

<svelte:head><title>{project?.name ?? 'Project'} · Masih Awam</title></svelte:head>

<AppShell crumbs={[{ label: 'Projects', href: '/app/projects' }, { label: project?.name ?? 'Project' }]} width="dense">
  {#if project === undefined}
    <h1 class="text-xl">We couldn't find that project</h1>
    <a class="link link-hover mt-4 inline-block font-medium text-primary" href="/app/projects">Back to projects</a>
  {:else}
    <div class="flex flex-wrap items-center gap-4">
      <ProjectTile name={project.name} id={project.id} class="size-10 text-lg" />
      <div class="min-w-0 flex-1">
        <h1 class="text-xl font-semibold tracking-tight">{project.name}</h1>
        {#if project.description !== ''}<p class="text-[var(--brand-text-secondary)]">{project.description}</p>{/if}
      </div>
      <button class="btn btn-primary btn-sm rounded-lg" type="button" onclick={newTask}><Icon name="plus" class="size-4" /> New task</button>
    </div>
    <div role="tablist" class="tabs tabs-border mt-6 border-b border-[var(--brand-border)]">
      {#each tabs as item (item.id)}
        <button role="tab" class="tab {tab === item.id ? 'tab-active font-medium' : ''}" aria-selected={tab === item.id} onclick={() => (tab = item.id)}>{item.label}</button>
      {/each}
    </div>
    <div class="mt-4">
      {#if tab === 'tasks'}<TasksTab {projectId} />{:else if tab === 'files'}{#key projectId}<FileBrowser {projectId} />{/key}{:else if tab === 'checks'}<ChecksTab {projectId} />{:else}<AboutTab {project} />{/if}
    </div>
  {/if}
</AppShell>
