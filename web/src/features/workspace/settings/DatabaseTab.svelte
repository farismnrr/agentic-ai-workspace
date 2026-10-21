<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '../../../shared/api/http';
  import Icon from '../../../shared/ui/Icon.svelte';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import FormField from '../components/FormField.svelte';
  import { getSharedDatabaseSettings, saveSharedDatabaseSettings, type SSLMode } from './database-api';
  import SettingsSection from './parts/SettingsSection.svelte';

  const sslModes: SSLMode[] = ['disable', 'allow', 'prefer', 'require', 'verify-ca', 'verify-full'];
  const sslModeHints: Record<SSLMode, string> = {
    disable: 'Do not encrypt the connection.',
    allow: 'Try a plain connection first, then TLS.',
    prefer: 'Try TLS first, then allow a plain connection.',
    require: 'Encrypt traffic without checking the server certificate.',
    'verify-ca': 'Check the server certificate chain, not its hostname.',
    'verify-full': 'Check the server certificate chain and hostname.'
  };
  type Removal = 'ca' | 'client';

  let host = $state('');
  let port = $state(5432);
  let database = $state('');
  let user = $state('');
  let password = $state('');
  let sslMode = $state<SSLMode>('prefer');
  let caCert = $state('');
  let clientCert = $state('');
  let clientKey = $state('');
  let passwordSet = $state(false);
  let caSet = $state(false);
  let clientCertSet = $state(false);
  let clientKeySet = $state(false);
  let loading = $state(true);
  let loadFailed = $state(false);
  let saving = $state(false);
  let message = $state('');
  let failed = $state(false);
  let pendingRemoval = $state<Removal | undefined>();

  onMount(async () => {
    try {
      const settings = await getSharedDatabaseSettings();
      host = settings.host;
      port = settings.port || 5432;
      database = settings.database;
      user = settings.user;
      sslMode = settings.sslMode;
      passwordSet = settings.passwordSet;
      caSet = settings.caSet;
      clientCertSet = settings.clientCertSet;
      clientKeySet = settings.clientKeySet;
    } catch {
      loadFailed = true;
      message = 'Could not load database settings.';
    } finally {
      loading = false;
    }
  });

  function errorMessage(error: unknown): string {
    if (error instanceof ApiError && error.code === 'invalid_settings') {
      return 'Check the connection details. Certificates must be valid PEM, and the client certificate and key must match.';
    }
    return 'Could not save these settings. Check the fields and try again.';
  }

  async function save(removal?: Removal): Promise<void> {
    saving = true;
    failed = false;
    message = '';
    try {
      const result = await saveSharedDatabaseSettings({
        host, port, database, user, password, sslMode, caCert, clientCert, clientKey,
        removeCaCert: removal === 'ca', removeClientCert: removal === 'client'
      });
      password = '';
      caCert = '';
      clientCert = '';
      clientKey = '';
      passwordSet = result.passwordSet;
      caSet = result.caSet;
      clientCertSet = result.clientCertSet;
      clientKeySet = result.clientKeySet;
      message = removal === 'ca' ? 'CA certificate removed.'
        : removal === 'client' ? 'Client certificate and key removed.' : 'Database settings saved.';
    } catch (error) {
      failed = true;
      message = errorMessage(error);
    } finally {
      saving = false;
    }
  }

  async function confirmRemoval(): Promise<void> {
    const removal = pendingRemoval;
    pendingRemoval = undefined;
    if (removal !== undefined) await save(removal);
  }

  function submit(event: SubmitEvent): void {
    event.preventDefault();
    void save();
  }
</script>

{#if loading}
  <div class="py-3" role="status"><span class="loading loading-spinner loading-sm" aria-label="Loading database settings"></span></div>
{:else if loadFailed}
  <div class="alert alert-error alert-soft" role="alert">{message}</div>
{:else}
  <SettingsSection title="Shared PostgreSQL" description="Projects can use this account to create a private database schema.">
    <form class="space-y-4 py-3" onsubmit={submit}>
      <div class="grid gap-3 sm:grid-cols-2">
        <FormField id="database-host" label="Host">
          {#snippet children({ describedby, invalid })}<input id="database-host" class="input input-sm w-full" aria-describedby={describedby} aria-invalid={invalid} bind:value={host} required />{/snippet}
        </FormField>
        <FormField id="database-port" label="Port">
          {#snippet children({ describedby, invalid })}<input id="database-port" class="input input-sm w-full" type="number" min="1" max="65535" aria-describedby={describedby} aria-invalid={invalid} bind:value={port} required />{/snippet}
        </FormField>
        <FormField id="database-name" label="Database">
          {#snippet children({ describedby, invalid })}<input id="database-name" class="input input-sm w-full" aria-describedby={describedby} aria-invalid={invalid} bind:value={database} required />{/snippet}
        </FormField>
        <FormField id="database-user" label="User">
          {#snippet children({ describedby, invalid })}<input id="database-user" class="input input-sm w-full" aria-describedby={describedby} aria-invalid={invalid} bind:value={user} required />{/snippet}
        </FormField>
        <FormField id="database-password" label="Password" optional={passwordSet}>
          {#snippet children({ describedby, invalid })}<input id="database-password" class="input input-sm w-full" type="password" autocomplete="new-password" placeholder={passwordSet ? 'Saved; leave blank to keep it' : ''} aria-describedby={describedby} aria-invalid={invalid} bind:value={password} required={!passwordSet} />{/snippet}
        </FormField>
        <FormField id="database-ssl-mode" label="SSL mode" hint={sslModeHints[sslMode]}>
          {#snippet children({ describedby, invalid })}
            <select id="database-ssl-mode" class="select select-sm w-full" bind:value={sslMode} aria-describedby={describedby} aria-invalid={invalid}>
              {#each sslModes as mode (mode)}<option value={mode}>{mode}</option>{/each}
            </select>
          {/snippet}
        </FormField>
      </div>

      <FormField id="database-ca-cert" label="CA certificate" optional hint="Paste a PEM certificate. Leave blank to keep any saved value.">
        {#snippet children({ describedby, invalid })}
          <div class="space-y-2">
            {#if caSet}
              <div class="flex items-center gap-2"><span class="badge badge-sm badge-success badge-soft">Saved</span><button class="btn btn-ghost btn-xs btn-square" type="button" aria-label="Remove saved CA certificate" disabled={saving} onclick={() => (pendingRemoval = 'ca')}><Icon name="trash" class="size-4" /></button></div>
            {/if}
            <textarea id="database-ca-cert" class="textarea textarea-sm w-full" rows="4" placeholder="-----BEGIN CERTIFICATE-----" bind:value={caCert} aria-describedby={describedby} aria-invalid={invalid}></textarea>
          </div>
        {/snippet}
      </FormField>

      <fieldset class="fieldset space-y-3">
        <legend class="fieldset-legend">Client certificate and key</legend>
        <p class="text-xs text-[var(--brand-text-secondary)]">Both are needed together. Leave blank to keep saved values.</p>
        {#if clientCertSet || clientKeySet}
          <div class="flex items-center gap-2"><span class="badge badge-sm badge-success badge-soft">Saved</span><button class="btn btn-ghost btn-xs btn-square" type="button" aria-label="Remove saved client certificate and key" disabled={saving} onclick={() => (pendingRemoval = 'client')}><Icon name="trash" class="size-4" /></button></div>
        {/if}
        <div class="grid gap-3 sm:grid-cols-2">
          <FormField id="database-client-cert" label="Client certificate" optional>
            {#snippet children({ describedby, invalid })}
              <textarea id="database-client-cert" class="textarea textarea-sm w-full" rows="4" placeholder="-----BEGIN CERTIFICATE-----" bind:value={clientCert} aria-describedby={describedby} aria-invalid={invalid}></textarea>
            {/snippet}
          </FormField>
          <FormField id="database-client-key" label="Client private key" optional>
            {#snippet children({ describedby, invalid })}
              <textarea id="database-client-key" class="textarea textarea-sm w-full" rows="4" placeholder="-----BEGIN PRIVATE KEY-----" bind:value={clientKey} aria-describedby={describedby} aria-invalid={invalid}></textarea>
            {/snippet}
          </FormField>
        </div>
      </fieldset>

      <div class="flex flex-wrap items-center gap-3">
        <button class="btn btn-primary btn-sm" type="submit" disabled={saving}>{saving ? 'Saving…' : 'Save database settings'}</button>
        {#if message !== ''}<p class="text-xs {failed ? 'text-error' : 'text-[var(--brand-text-secondary)]'}" role={failed ? 'alert' : 'status'}>{message}</p>{/if}
      </div>
    </form>
  </SettingsSection>
{/if}

<ConfirmModal
  open={pendingRemoval !== undefined}
  title={pendingRemoval === 'ca' ? 'Remove saved CA certificate?' : 'Remove saved client certificate and key?'}
  body="This removes the saved certificate material from shared database settings."
  confirmLabel="Remove"
  cancelLabel="Keep it"
  oncancel={() => (pendingRemoval = undefined)}
  onconfirm={confirmRemoval}
/>
