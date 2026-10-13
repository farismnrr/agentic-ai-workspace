<script lang="ts">
  import LandingPage from '../features/landing/LandingPage.svelte';
  import LoginPage from '../features/auth/LoginPage.svelte';
  import RegisterPage from '../features/auth/RegisterPage.svelte';
  import { interceptLinkClick, route, syncFromLocation } from './router';
</script>

<svelte:window onclick={interceptLinkClick} onpopstate={syncFromLocation} />

{#if $route === 'login'}
  <LoginPage />
{:else if $route === 'register'}
  <RegisterPage />
{:else}
  <LandingPage />
{/if}
