<script lang="ts">
  import Icon from '../../../shared/ui/Icon.svelte';
  import { kindLabel, modelsCopy } from '../content/models';
  import { programs } from '../content/subscriptions';
  import { describeProviderError } from '../logic/provider-errors';
  import { setProviderEnabled, verifyProvider, type Provider } from '../logic/providers-api';

  interface Props {
    provider: Provider;
    onchange: () => void;
    onremove: () => void;
    onerror: (message: string) => void;
  }

  const { provider, onchange, onremove, onerror }: Props = $props();
  let busy = $state(false);
  const plan = $derived(provider.kind === 'plan' ? programs[provider.id.slice('plan:'.length) as keyof typeof programs] : undefined);

  async function run(action: () => Promise<unknown>): Promise<void> {
    busy = true;
    try {
      await action();
    } catch (error) {
      onerror(describeProviderError(error));
    } finally {
      busy = false;
      onchange();
    }
  }
</script>

<li class="flex flex-wrap items-center gap-3 py-3">
  <span class="grid size-8 shrink-0 place-items-center rounded-lg bg-base-300 text-sm font-medium" aria-hidden="true">{provider.name.slice(0, 1)}</span>
  <span class="flex min-w-0 flex-1 flex-col leading-tight">
    <span class="font-medium">{provider.name} <span class="font-normal text-xs text-[var(--brand-text-tertiary)]">· {plan === undefined ? kindLabel(provider.kind) : plan.badge}</span></span>
    {#if plan === undefined}
      <code class="truncate text-xs text-[var(--brand-text-secondary)]">{provider.keyHint} · {provider.baseUrl}</code>
    {:else}
      <span class="text-xs text-[var(--brand-text-secondary)]">{plan.connected}</span>
    {/if}
  </span>
  {#if plan !== undefined}
    <span class="flex items-center gap-1 text-xs"><Icon name="check-circle" class="size-4 text-success" /> Signed in</span>
  {:else if provider.verified}
    <span class="flex items-center gap-1 text-xs"><Icon name="check-circle" class="size-4 text-success" /> Verified</span>
    <button class="btn btn-ghost btn-xs" type="button" disabled={busy} title={modelsCopy.refreshTip} onclick={() => run(() => verifyProvider(provider.id))}>Refresh</button>
  {:else}
    <button class="btn btn-outline btn-xs" type="button" disabled={busy} onclick={() => run(() => verifyProvider(provider.id))}>Verify key</button>
  {/if}
  <input
    type="checkbox"
    class="toggle toggle-primary toggle-sm"
    aria-label="{provider.name} on or off"
    checked={provider.enabled}
    disabled={busy}
    onchange={(event) => run(() => setProviderEnabled(provider.id, event.currentTarget.checked))}
  />
  <button class="btn btn-ghost btn-xs btn-square" type="button" aria-label="{plan === undefined ? 'Remove' : 'Sign out of'} {provider.name}" onclick={onremove}><Icon name="trash" class="size-4" /></button>
</li>
