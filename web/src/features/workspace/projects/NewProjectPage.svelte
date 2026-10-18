<script lang="ts">
  import { navigate } from '../../../app/router';
  import Icon from '../../../shared/ui/Icon.svelte';
  import { templatePreview } from '../assets/template-previews';
  import { projectsCopy } from '../content/projects-copy';
  import { emptyStart, githubStart, newProjectCopy, templateStarts } from '../content/templates-copy';
  import { createNewProject } from '../logic/create-project';
  import { loadGitHub } from '../logic/github-api';
  import { describeProjectError } from '../logic/project-errors';
  import { refresh } from '../logic/workspace-store';
  import AppShell from '../shell/AppShell.svelte';
  import StartDetails from './StartDetails.svelte';
  import StartGallery from './StartGallery.svelte';
  import StartWhere from './StartWhere.svelte';

  type Step = 'start' | 'where' | 'details';

  let step = $state<Step>('start');
  let github = $state<'off' | 'connect' | 'ready'>('off');
  /** '' is an empty project; otherwise the name of the template to start from. */
  let template = $state('');
  let source = $state<'empty' | 'github' | 'new'>('empty');
  let name = $state('');
  let description = $state('');
  let repo = $state('');
  let repoName = $state('');
  let isPrivate = $state(true);
  let busy = $state(false);
  let error = $state('');

  $effect(() => {
    void loadGitHub().then((status) => { github = status.connected ? 'ready' : status.configured ? 'connect' : 'off'; }, () => { github = 'off'; });
  });

  const chosen = $derived(source === 'github' ? githubStart : (templateStarts.find((start) => start.id === template) ?? emptyStart));

  function start(id: string): void {
    error = '';
    if (id === githubStart.id) {
      source = 'github';
      template = '';
      step = 'details';
      return;
    }
    template = id;
    source = 'empty';
    step = 'where';
  }

  function where(chosenSource: 'empty' | 'new'): void {
    source = chosenSource;
    step = 'details';
  }

  async function submit(): Promise<void> {
    busy = true;
    error = '';
    try {
      const project = await createNewProject({ name, description, source, template, repo, repoName, isPrivate });
      await refresh();
      navigate(`/app/projects/${project.id}`);
    } catch (failure) {
      error = describeProjectError(failure);
    } finally {
      busy = false;
    }
  }
</script>

<svelte:head><title>{newProjectCopy.title} · Masih Awam</title></svelte:head>

<AppShell crumbs={[{ label: 'Projects', href: '/app/projects' }, { label: newProjectCopy.title }]} width="dense">
  <h1 class="text-xl font-semibold tracking-tight">{newProjectCopy.title}</h1>
  <p class="mt-1 text-[var(--brand-text-secondary)]">{projectsCopy.intro}</p>

  {#if step !== 'start'}
    <div class="mt-6 flex max-w-xl items-center gap-3 rounded-lg border border-[var(--brand-border)] bg-base-100 p-2">
      {#if template !== '' && templatePreview(template) !== undefined}
        <img class="h-10 w-16 rounded object-cover object-top" src={templatePreview(template)} alt="" width="64" height="40" />
      {:else}
        <Icon name={source === 'github' ? 'folder' : 'plus'} class="mx-4 size-5 text-primary" />
      {/if}
      <p class="min-w-0 flex-1 text-sm"><span class="text-[var(--brand-text-secondary)]">{newProjectCopy.chosen}</span> <span class="font-medium">{chosen.title}</span></p>
      <button class="btn btn-ghost btn-sm" type="button" onclick={() => { step = 'start'; }}>{newProjectCopy.change}</button>
    </div>
  {/if}

  <div class="mt-8">
    {#if step === 'start'}
      <StartGallery {github} onpick={start} />
    {:else if step === 'where'}
      <StartWhere {github} onpick={where} />
    {:else}
      <StartDetails {source} bind:name bind:description bind:repo bind:repoName bind:isPrivate {busy} {error} onsubmit={() => void submit()} oncancel={() => { step = 'start'; }} />
    {/if}
  </div>
</AppShell>
