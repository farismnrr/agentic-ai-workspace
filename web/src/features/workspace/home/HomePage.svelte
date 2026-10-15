<script lang="ts">
  import Marker from '../../../shared/ui/Marker.svelte';
  import { session } from '../../../shared/session/session';
  import { greeting } from '../content/user';
  import AppShell from '../shell/AppShell.svelte';
  import Composer from './Composer.svelte';
  import NeedsYou from './NeedsYou.svelte';
  import RunningNow from './RunningNow.svelte';
</script>

<svelte:head><title>Home · Masih Awam</title></svelte:head>

<AppShell crumbs={[{ label: 'Home' }]} width="spacious">
  <h1 class="font-display! text-4xl font-medium leading-tight tracking-tight">{greeting(new Date().getHours())}, <Marker>{$session.status === 'signed-in' ? $session.user.name : ''}.</Marker></h1>
  <p class="mb-6 mt-1 text-base text-[var(--brand-text-secondary)]">What do you need help with today?</p>
  <Composer />
  <div class="mt-10 grid grid-cols-1 items-start gap-6 lg:grid-cols-2"><NeedsYou /><RunningNow /></div>
</AppShell>
