<script lang="ts">
  import { navigate } from '../../../app/router';
  import { session, signOut } from '../../../shared/session/session';
  import Icon from '../../../shared/ui/Icon.svelte';

  interface Props {
    collapsed: boolean;
  }

  const { collapsed }: Props = $props();
  const user = $derived($session.status === 'signed-in' ? $session.user : { name: '', email: '' });
  let open = $state(false);
  let root: HTMLDetailsElement | undefined = $state();

  const close = (): void => {
    open = false;
  };

  /** A <details> stays open on its own, so close it on an outside click like every other menu. */
  function clickAway(event: MouseEvent): void {
    if (root?.open === true && event.target instanceof Node && !root.contains(event.target)) close();
  }

  function onkeydown(event: KeyboardEvent): void {
    if (event.key !== 'Escape' || root?.open !== true) return;
    close();
    root.querySelector('summary')?.focus();
  }

  async function leave(): Promise<void> {
    close();
    await signOut();
    navigate('/login');
  }
</script>

<svelte:window onclick={clickAway} {onkeydown} />

<details bind:this={root} bind:open class="dropdown dropdown-top w-full">
  <summary class="flex min-h-10 cursor-pointer list-none items-center gap-2.5 rounded-lg px-2 hover:bg-base-100/70">
    <span class="grid size-6 shrink-0 place-items-center rounded-full bg-primary text-xs font-medium text-primary-content" aria-hidden="true">{user.name.slice(0, 1).toUpperCase()}</span>
    <span class="flex-1 truncate font-medium {collapsed ? 'lg:hidden' : ''}">{user.name}</span>
    <Icon name="chevron-down" class="size-4 transition-transform motion-reduce:transition-none {open ? 'rotate-180' : ''} {collapsed ? 'lg:hidden' : ''}" />
  </summary>
  <div class="dropdown-content z-30 mb-2 w-full min-w-56 rounded-xl border border-[var(--brand-border)] bg-base-100 p-1.5 shadow-[var(--brand-shadow)]">
    <div class="flex items-center gap-2.5 px-2 py-2">
      <span class="grid size-8 shrink-0 place-items-center rounded-full bg-primary text-sm font-medium text-primary-content" aria-hidden="true">{user.name.slice(0, 1).toUpperCase()}</span>
      <span class="flex min-w-0 flex-col leading-tight">
        <span class="truncate font-medium">{user.name}</span>
        <span class="truncate text-xs text-[var(--brand-text-secondary)]">{user.email}</span>
      </span>
    </div>
    <ul class="menu w-full border-t border-[var(--brand-border)] p-0 pt-1">
      <li><a href="/app/settings/account" onclick={close}><Icon name="user" class="size-4" />Account</a></li>
      <li><a href="/app/settings/usage" onclick={close}><Icon name="clock" class="size-4" />Usage</a></li>
    </ul>
    <ul class="menu mt-1 w-full border-t border-[var(--brand-border)] p-0 pt-1">
      <li><button type="button" onclick={leave}><Icon name="log-out" class="size-4" />Sign out</button></li>
    </ul>
  </div>
</details>
