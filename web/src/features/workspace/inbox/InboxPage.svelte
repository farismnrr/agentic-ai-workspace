<script lang="ts">
  import { navigate, route } from '../../../app/router';
  import { inboxItems } from '../logic/derive';
  import { tasks } from '../logic/workspace-store';
  import AppShell from '../shell/AppShell.svelte';
  import InboxDetail from './InboxDetail.svelte';
  import InboxList from './InboxList.svelte';

  const items = $derived(inboxItems($tasks));
  const routeId = $derived($route.name === 'inbox' ? $route.taskId : undefined);
  const selected = $derived(items.find((item) => item.id === routeId));
  const desktopPick = $derived(selected ?? items[0]);

  function onkeydown(event: KeyboardEvent): void {
    const target = event.target;
    if (target instanceof HTMLElement && ['INPUT', 'TEXTAREA'].includes(target.tagName)) return;
    const step = event.key === 'j' ? 1 : event.key === 'k' ? -1 : 0;
    if (step === 0 || desktopPick === undefined) return;
    const next = items[items.indexOf(desktopPick) + step];
    if (next !== undefined) navigate(`/app/inbox/${next.id}`);
  }
</script>

<svelte:head><title>Inbox · Masih Awam</title></svelte:head>
<svelte:window {onkeydown} />

<AppShell crumbs={[{ label: 'Inbox' }]} width="split">
  {#if items.length === 0}
    <div class="grid flex-1 place-items-center p-8 text-center"><div><h1 class="text-xl">You're all caught up</h1><p class="mt-2 text-[var(--brand-text-secondary)]">Nothing is waiting for you right now.</p></div></div>
  {:else}
    <div class="flex flex-1 lg:h-[calc(100dvh-3rem)]">
      <section class="w-full flex-col overflow-y-auto border-r border-[var(--brand-border)] lg:flex lg:w-96 lg:flex-none {routeId === undefined ? 'flex' : 'hidden'}" aria-label="Inbox list">
        <div class="flex items-baseline justify-between px-4 py-3">
          <h1 class="font-sans text-lg font-semibold">Inbox</h1>
          <span class="text-xs text-[var(--brand-text-tertiary)]"><kbd class="kbd kbd-xs">j</kbd> <kbd class="kbd kbd-xs">k</kbd> to move</span>
        </div>
        <InboxList {items} selectedId={desktopPick?.id} />
      </section>
      <section class="min-w-0 flex-1 overflow-y-auto bg-base-200 lg:block {routeId === undefined ? 'hidden' : 'block'}" aria-label="Selected item">
        {#if desktopPick !== undefined}{#key desktopPick.id}<InboxDetail item={desktopPick} />{/key}{/if}
      </section>
    </div>
  {/if}
</AppShell>
