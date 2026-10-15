<script lang="ts">
  import type { SettingsTab } from '../../../app/router';
  import AppShell from '../shell/AppShell.svelte';
  import AccountTab from './AccountTab.svelte';
  import ActivityTab from './ActivityTab.svelte';
  import ConnectedAppsTab from './ConnectedAppsTab.svelte';
  import IntegrationsTab from './IntegrationsTab.svelte';
  import ModelsTab from './ModelsTab.svelte';
  import NotificationsTab from './NotificationsTab.svelte';
  import PersonaTab from './PersonaTab.svelte';
  import SettingsNav from './SettingsNav.svelte';
  import { currentLabel } from './settings-nav';
  import SpacesTab from './SpacesTab.svelte';
  import UsageTab from './UsageTab.svelte';

  interface Props {
    tab: SettingsTab;
  }

  const { tab }: Props = $props();
  const label = $derived(currentLabel(tab));
</script>

<svelte:head><title>Settings · Masih Awam</title></svelte:head>

<AppShell crumbs={[{ label: 'Settings', href: '/app/settings' }, { label }]} width="dense">
  <div class="grid items-start gap-6 md:grid-cols-[12rem_1fr]">
    <div class="md:sticky md:top-16"><h1 class="mb-3 text-xl font-semibold">Settings</h1><SettingsNav {tab} /></div>
    <div class="min-w-0 max-w-4xl">
      {#if tab === 'models'}<ModelsTab />{:else if tab === 'persona'}<PersonaTab />{:else if tab === 'spaces'}<SpacesTab />{:else if tab === 'connected'}<ConnectedAppsTab />{:else if tab === 'integrations'}<IntegrationsTab />{:else if tab === 'activity'}<ActivityTab />{:else if tab === 'usage'}<UsageTab />{:else if tab === 'notifications'}<NotificationsTab />{:else}<AccountTab />{/if}
    </div>
  </div>
</AppShell>
