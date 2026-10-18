<script lang="ts">
  import { onMount } from 'svelte';
  import LandingPage from '../features/landing/LandingPage.svelte';
  import ErrorPage from '../features/errors/ErrorPage.svelte';
  import LoginPage from '../features/auth/LoginPage.svelte';
  import RegisterPage from '../features/auth/RegisterPage.svelte';
  import HomePage from '../features/workspace/home/HomePage.svelte';
  import InboxPage from '../features/workspace/inbox/InboxPage.svelte';
  import ProjectDetailPage from '../features/workspace/projects/ProjectDetailPage.svelte';
  import NewProjectPage from '../features/workspace/projects/NewProjectPage.svelte';
  import ProjectsPage from '../features/workspace/projects/ProjectsPage.svelte';
  import SitesPage from '../features/workspace/sites/SitesPage.svelte';
  import SettingsPage from '../features/workspace/settings/SettingsPage.svelte';
  import TaskPage from '../features/workspace/task/TaskPage.svelte';
  import ConsentPage from '../features/auth/ConsentPage.svelte';
  import { loadSession, session } from '../shared/session/session';
  import { nextPath } from '../shared/session/next-path';
  import { guestOnly, loginUrl, requiresSession } from './guard';
  import { goTo, interceptLinkClick, navigate, route, syncFromLocation } from './router';

  const gated = $derived(requiresSession($route));
  const guest = $derived(guestOnly($route));

  onMount(() => void loadSession());

  $effect(() => {
    if (gated && $session.status === 'anonymous') navigate(loginUrl(window.location.pathname + window.location.search), true);
  });

  // Already signed in (or just signed in on this page): leave the sign-in and register pages for where the person
  // was going, or the dashboard. Replacing the entry keeps Back from returning to a page they cannot use.
  $effect(() => {
    if (guest && $session.status === 'signed-in') goTo(nextPath(window.location.search), true);
  });
</script>

<svelte:window onclick={interceptLinkClick} onpopstate={syncFromLocation} />

{#if (gated && $session.status !== 'signed-in') || (guest && $session.status !== 'anonymous')}
  <main class="grid min-h-dvh place-items-center" aria-busy="true"><span class="loading loading-spinner"></span></main>
{:else if $route.name === 'login'}
  <LoginPage />
{:else if $route.name === 'register'}
  <RegisterPage />
{:else if $route.name === 'consent'}
  <ConsentPage />
{:else if $route.name === 'home'}
  <HomePage />
{:else if $route.name === 'inbox'}
  <InboxPage />
{:else if $route.name === 'projects'}
  <ProjectsPage />
{:else if $route.name === 'new-project'}
  <NewProjectPage />
{:else if $route.name === 'sites'}
  <SitesPage />
{:else if $route.name === 'project'}
  <ProjectDetailPage projectId={$route.projectId} />
{:else if $route.name === 'task'}
  <TaskPage projectId={$route.projectId} taskId={$route.taskId} />
{:else if $route.name === 'settings'}
  <SettingsPage tab={$route.tab} />
{:else if $route.name === 'not-found' || $route.name === 'forbidden'}
  <ErrorPage kind={$route.name} />
{:else}
  <LandingPage />
{/if}
