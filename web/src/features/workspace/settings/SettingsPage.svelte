<script lang="ts">
  import type { SettingsTab } from '../../../app/router';
  import SettingsFrame from './SettingsFrame.svelte';
  import AccountTab from './AccountTab.svelte';
  import ActivityTab from './ActivityTab.svelte';
  import ConnectedAppsTab from './ConnectedAppsTab.svelte';
  import DatabaseTab from './DatabaseTab.svelte';
  import IntegrationsTab from './IntegrationsTab.svelte';
  import ModelsTab from './ModelsTab.svelte';
  import NotificationsTab from './NotificationsTab.svelte';
  import PersonaTab from './PersonaTab.svelte';
  import { currentLabel } from './settings-nav';
  import SpacesTab from './SpacesTab.svelte';
  import TeamTab from './TeamTab.svelte';
  import UsageTab from './UsageTab.svelte';

  interface Props {
    tab: SettingsTab;
  }

  const { tab }: Props = $props();
  const label = $derived(currentLabel(tab));
</script>

<svelte:head><title>Settings · Masih Awam</title></svelte:head>

<SettingsFrame crumbs={[{ label: 'Settings', href: '/app/settings' }, { label }]}>
  {#if tab === 'models'}<ModelsTab />{:else if tab === 'persona'}<PersonaTab />{:else if tab === 'team'}<TeamTab />{:else if tab === 'spaces'}<SpacesTab />{:else if tab === 'connected'}<ConnectedAppsTab />{:else if tab === 'integrations'}<IntegrationsTab />{:else if tab === 'activity'}<ActivityTab />{:else if tab === 'usage'}<UsageTab />{:else if tab === 'notifications'}<NotificationsTab />{:else if tab === 'database'}<DatabaseTab />{:else}<AccountTab />{/if}
</SettingsFrame>
