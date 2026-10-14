<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import { navigate } from '../../../app/router';
  import { session, signOut } from '../../../shared/session/session';
  import { plan } from '../content/user';

  interface Props {
    collapsed: boolean;
  }

  const { collapsed }: Props = $props();
  const user = $derived($session.status === 'signed-in' ? $session.user : { name: '', email: '' });

  async function leave(): Promise<void> {
    await signOut();
    navigate('/login');
  }
</script>

<details class="dropdown dropdown-top w-full">
  <summary class="flex min-h-10 cursor-pointer list-none items-center gap-2.5 rounded-lg px-2 hover:bg-base-100/70">
    <span class="grid size-6 shrink-0 place-items-center rounded-full bg-base-300 text-xs font-medium" aria-hidden="true">{user.name.slice(0, 1)}</span>
    <span class="flex-1 truncate font-medium {collapsed ? 'lg:hidden' : ''}">{user.name}</span>
    <Icon name="chevron-down" class="size-4 {collapsed ? 'lg:hidden' : ''}" />
  </summary>
  <div class="dropdown-content z-30 mb-2 w-60 rounded-xl border border-[var(--brand-border)] bg-base-100 p-2 shadow-lg">
    <p class="px-2 pt-1 text-xs text-[var(--brand-text-secondary)]">{user.email}</p>
    <div class="px-2 pb-2 pt-2">
      <div class="mb-1 flex justify-between text-xs"><span>{plan.name}</span><span class="text-[var(--brand-text-secondary)]">{plan.used} of {plan.limit} tasks</span></div>
      <progress class="progress progress-primary h-1" value={plan.used} max={plan.limit} aria-label="Tasks used this month"></progress>
    </div>
    <ul class="menu p-0">
      <li><a href="/app/settings/account">Account</a></li>
      <li><a href="/app/settings/usage">Usage</a></li>
      <li><button type="button" onclick={leave}>Sign out</button></li>
    </ul>
  </div>
</details>
