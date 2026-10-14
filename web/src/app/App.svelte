<script lang="ts">
  import { onMount } from 'svelte';
  import LandingPage from '../features/landing/LandingPage.svelte';
  import LoginPage from '../features/auth/LoginPage.svelte';
  import RegisterPage from '../features/auth/RegisterPage.svelte';
  import HomePage from '../features/workspace/home/HomePage.svelte';
  import InboxPage from '../features/workspace/inbox/InboxPage.svelte';
  import ProjectDetailPage from '../features/workspace/projects/ProjectDetailPage.svelte';
  import ProjectsPage from '../features/workspace/projects/ProjectsPage.svelte';
  import SettingsPage from '../features/workspace/settings/SettingsPage.svelte';
  import TaskPage from '../features/workspace/task/TaskPage.svelte';
  import ConsentPage from '../features/auth/ConsentPage.svelte';
  import { loadSession, session } from '../shared/session/session';
  import { loginUrl, requiresSession } from './guard';
  import { interceptLinkClick, navigate, route, syncFromLocation } from './router';

  const gated = $derived(requiresSession($route));

  onMount(() => void loadSession());

  $effect(() => {
    if (gated && $session.status === 'anonymous') navigate(loginUrl(window.location.pathname + window.location.search), true);
  });
</script>

<svelte:window onclick={interceptLinkClick} onpopstate={syncFromLocation} />

{#if gated && $session.status !== 'signed-in'}
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
{:else if $route.name === 'project'}
  <ProjectDetailPage projectId={$route.projectId} />
{:else if $route.name === 'task'}
  <TaskPage projectId={$route.projectId} taskId={$route.taskId} />
{:else if $route.name === 'settings'}
  <SettingsPage tab={$route.tab} />
{:else}
  <LandingPage />
{/if}
