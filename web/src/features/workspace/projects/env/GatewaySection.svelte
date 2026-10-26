<script lang="ts">
  import { onMount } from 'svelte';
  import FormField from '../../components/FormField.svelte';
  import SettingsSection from '../../settings/parts/SettingsSection.svelte';
  import { clearGateway, getGateway, setGateway, type GatewayView } from './gateway-api';

  interface Props { projectId: string }
  const { projectId }: Props = $props();
  let view = $state<GatewayView>();
  let loading = $state(true), loadFailed = $state(false), busy = $state(false);
  let message = $state(''), messageError = $state(false);

  const chosen = $derived(view?.options.find((item) => item.id === view?.selected));
  const options = $derived(view?.options ?? []);
  const noOptions = $derived(view !== undefined && options.length === 0);
  onMount(() => { void load(); });

  async function load(): Promise<void> {
    loading = true; loadFailed = false;
    try { view = await getGateway(projectId); }
    catch { loadFailed = true; }
    finally { loading = false; }
  }
  async function choose(id: string): Promise<void> {
    busy = true; message = '';
    try {
      if (id === '') await clearGateway(projectId); else await setGateway(projectId, id);
      view = await getGateway(projectId);
      message = id === '' ? 'This project no longer reaches a model gateway.' : 'This project can now reach that gateway.';
      messageError = false;
    } catch { message = 'Could not change this project’s model gateway. Try again.'; messageError = true; }
    finally { busy = false; }
  }
</script>

<SettingsSection title="Model gateway" description="Let this project's code use an OpenAI-compatible local API with your own API key. Account relays are unsupported.">
  {#if loading}
    <div class="py-3" role="status"><span class="loading loading-spinner loading-sm" aria-label="Loading model gateway"></span></div>
  {:else if loadFailed}
    <div class="py-3 alert alert-error alert-soft" role="alert"><span>Could not load the model gateway.</span><button class="btn btn-sm" type="button" onclick={() => { void load(); }}>Try again</button></div>
  {:else if view !== undefined}
    <div class="space-y-3 py-4">
      <FormField id="project-gateway" label="Gateway" hint={noOptions ? 'Add an OpenAI-compatible provider that points to this computer in Settings → Models first.' : undefined}>
        {#snippet children({ describedby })}
          <select id="project-gateway" class="select select-sm w-full" value={view?.selected ?? ''} aria-describedby={describedby} disabled={busy || noOptions} onchange={(event) => { void choose(event.currentTarget.value); }}>
            <option value="">None</option>
            {#each options as item (item.id)}<option value={item.id}>{item.name} ({item.baseUrl})</option>{/each}
          </select>
        {/snippet}
      </FormField>
      {#if chosen !== undefined}
        <p class="text-sm text-[var(--brand-text-secondary)]">The code gets <code>OPENAI_BASE_URL</code>{chosen.baseUrl === '' ? '' : ` (${chosen.baseUrl})`}, and <code>OPENAI_API_KEY</code> when the provider has a key. Only that address is reachable.</p>
      {/if}
      {#if message !== ''}<p class="text-xs" class:text-error={messageError} role={messageError ? 'alert' : 'status'}>{message}</p>{/if}
    </div>
  {/if}
</SettingsSection>
