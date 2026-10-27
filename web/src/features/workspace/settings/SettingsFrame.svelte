<script lang="ts">
  import type { Snippet } from 'svelte';
  import { route } from '../../../app/router';
  import type { Crumb } from '../shell/shell-state';
  import AppShell from '../shell/AppShell.svelte';
  import { session } from '../../../shared/session/session';
  import SettingsNav from './SettingsNav.svelte';
  import { activeSettingsItem, currentLabel } from './settings-nav';

  interface Props {
    crumbs: Crumb[];
    children: Snippet;
  }

  const { crumbs, children }: Props = $props();
  const admin = $derived($session.status === 'signed-in' && $session.user.role === 'admin');
  const active = $derived(activeSettingsItem($route) ?? 'models');
  const label = $derived(currentLabel(active));
</script>

<AppShell {crumbs} width="dense">
  <div class="grid min-w-0 grid-cols-1 items-start gap-6 md:grid-cols-[12rem_minmax(0,1fr)]">
    <aside class="min-w-0 md:sticky md:top-16">
      <h1 class="mb-3 text-xl font-semibold">Settings</h1>
      <SettingsNav {admin} {active} />
    </aside>
    <section aria-label={label} class="min-w-0 max-w-4xl">
      {@render children()}
    </section>
  </div>
</AppShell>
