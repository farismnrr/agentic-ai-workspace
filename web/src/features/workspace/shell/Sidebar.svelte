<script lang="ts">
  import { route } from '../../../app/router';
  import LogoMark from '../../../shared/ui/LogoMark.svelte';
  import { inboxItems } from '../logic/derive';
  import { runningSites } from '../logic/sites';
  import { sites, tasks } from '../logic/workspace-store';
  import NavItem from './NavItem.svelte';
  import { paletteOpen, sidebarCollapsed } from './shell-state';
  import SidebarProjects from './SidebarProjects.svelte';
  import SidebarPinned from './SidebarPinned.svelte';
  import SidebarRecent from './SidebarRecent.svelte';
  import SidebarSites from './SidebarSites.svelte';
  import UserMenu from './UserMenu.svelte';

  const collapsed = $derived($sidebarCollapsed);
  const name = $derived($route.name);
  const waiting = $derived(inboxItems($tasks).length);
  const live = $derived(runningSites($sites).length);
  const inProjects = $derived(name === 'projects' || name === 'project');
</script>

<nav class="flex h-full min-h-dvh w-64 flex-col border-r border-[var(--brand-border)] bg-[var(--brand-panel)] p-2 {collapsed ? 'lg:w-14' : ''}" aria-label="Workspace">
  <a class="mb-2 flex min-h-9 items-center gap-2.5 px-2" href="/app" aria-label="Masih Awam home">
    <LogoMark class="size-6 shrink-0" />
    <span class="font-semibold {collapsed ? 'lg:hidden' : ''}">Masih Awam</span>
  </a>
  <NavItem href="/app" icon="pencil" label="New task" {collapsed} active={name === 'home'} />
  <NavItem icon="search" label="Search" {collapsed} hint="Ctrl K" onclick={() => { paletteOpen.set(true); }} />
  <NavItem href="/app/inbox" icon="inbox" label="Inbox" {collapsed} active={name === 'inbox'} badge={waiting} />
  <NavItem href="/app/tasks" icon="check-square" label="Tasks" {collapsed} active={name === 'tasks'} />
  <NavItem href="/app/archive" icon="box" label="Archive" {collapsed} active={name === 'archive'} />
  <NavItem href="/app/projects" icon="folder" label="All projects" {collapsed} active={inProjects} />
  <NavItem href="/app/sites" icon="globe" label="Sites" {collapsed} active={name === 'sites'} badge={live} />
  <NavItem href="/app/extensions" icon="puzzle" label="Extensions" {collapsed} active={name === 'extensions'} />
  <div class="min-h-0 flex-1 overflow-y-auto {collapsed ? 'lg:hidden' : ''}">
    <SidebarSites />
    <SidebarPinned />
    <SidebarRecent />
    <SidebarProjects />
  </div>
  <div class="mt-auto flex flex-col gap-0.5 border-t border-[var(--brand-border)] pt-2 {collapsed ? 'lg:mt-auto' : ''}">
    <NavItem href="/app/settings" icon="settings" label="Settings" {collapsed} active={name === 'settings'} />
    <UserMenu {collapsed} />
  </div>
</nav>
