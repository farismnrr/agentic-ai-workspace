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
  import StartSource from './StartSource.svelte';
  import StartWhere from './StartWhere.svelte';

  type Step = 'source' | 'template' | 'where' | 'details';

  let step = $state<Step>('source');
  let github = $state<'loading' | 'off' | 'connect' | 'ready'>('loading');
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

  // Without a GitHub App on the server there is only one way to start, so the first question is skipped.
  $effect(() => {
    if (github === 'off' && step === 'source') step = 'template';
  });

  const chosen = $derived(source === 'github' ? githubStart : (templateStarts.find((start) => start.id === template) ?? emptyStart));

  function pickSource(chosenSource: 'template' | 'github'): void {
    error = '';
    template = '';
    source = chosenSource === 'github' ? 'github' : 'empty';
    step = chosenSource === 'github' ? 'details' : 'template';
  }

  function start(id: string): void {
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

  {#if step === 'where' || step === 'details'}
    <div class="mt-6 flex max-w-xl items-center gap-3 rounded-lg border border-[var(--brand-border)] bg-base-100 p-2">
      {#if template !== '' && templatePreview(template) !== undefined}
        <img class="h-10 w-16 rounded object-cover object-top" src={templatePreview(template)} alt="" width="64" height="40" />
      {:else}
        <Icon name={source === 'github' ? 'folder' : 'plus'} class="mx-4 size-5 text-primary" />
      {/if}
      <p class="min-w-0 flex-1 text-sm"><span class="text-[var(--brand-text-secondary)]">{newProjectCopy.chosen}</span> <span class="font-medium">{chosen.title}</span></p>
      <button class="btn btn-ghost btn-sm" type="button" onclick={() => { step = source === 'github' ? 'source' : 'template'; }}>{newProjectCopy.change}</button>
    </div>
  {/if}

  <div class="mt-8">
    {#if step === 'source'}
      {#if github === 'ready' || github === 'connect'}<StartSource {github} onpick={pickSource} />{/if}
    {:else if step === 'template'}
      <StartGallery onpick={start} />
      {#if github === 'ready' || github === 'connect'}
        <button class="btn btn-ghost btn-sm mt-6" type="button" onclick={() => { step = 'source'; }}><Icon name="arrow-left" class="size-4" /> {newProjectCopy.back}</button>
      {/if}
    {:else if step === 'where'}
      <StartWhere github={github === 'loading' ? 'off' : github} onpick={where} />
    {:else}
      <StartDetails {source} bind:name bind:description bind:repo bind:repoName bind:isPrivate {busy} {error} onsubmit={() => void submit()} oncancel={() => { navigate('/app/projects'); }} />
    {/if}
  </div>
</AppShell>
