<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '../../../shared/api/http';
  import Icon from '../../../shared/ui/Icon.svelte';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import FormField from '../components/FormField.svelte';
  import { getSharedDatabaseSettings, saveSharedDatabaseSettings, sslModeForTLSChoice, tlsChoiceFromSettings, type TLSChoice } from './database-api';
  import SettingsSection from './parts/SettingsSection.svelte';

  const maxCertificateBytes = 16 * 1024;
  const certificateAccept = '.pem,.crt,.cer,.key';
  type CertificateField = 'ca' | 'clientCert' | 'clientKey';
  type Removal = 'ca' | 'client';

  let host = $state(''), database = $state(''), user = $state(''), password = $state('');
  let port = $state(5432);
  let tlsChoice = $state<TLSChoice>('none');
  let caCert = $state(''), clientCert = $state(''), clientKey = $state('');
  let caFileName = $state(''), clientCertFileName = $state(''), clientKeyFileName = $state('');
  let caError = $state(''), clientCertError = $state(''), clientKeyError = $state('');
  let passwordSet = $state(false), caSet = $state(false), clientCertSet = $state(false), clientKeySet = $state(false);
  let loading = $state(true), loadFailed = $state(false), saving = $state(false), failed = $state(false);
  let message = $state('');
  let pendingRemoval = $state<Removal | undefined>();

  onMount(async () => {
    try {
      const settings = await getSharedDatabaseSettings();
      host = settings.host;
      port = settings.port || 5432;
      database = settings.database;
      user = settings.user;
      tlsChoice = settings.configured ? tlsChoiceFromSettings(settings.sslMode, settings.clientCertSet) : 'none';
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
    if (error instanceof ApiError && error.code === 'invalid_settings') return 'Check the connection details and certificate files, then try again.';
    return 'Could not save these settings. Check the fields and try again.';
  }
  function setCertificate(field: CertificateField, value: string): void {
    if (field === 'ca') caCert = value; else if (field === 'clientCert') clientCert = value; else clientKey = value;
  }
  function setFileName(field: CertificateField, value: string): void {
    if (field === 'ca') caFileName = value; else if (field === 'clientCert') clientCertFileName = value; else clientKeyFileName = value;
  }
  function setFieldError(field: CertificateField, value: string): void {
    if (field === 'ca') caError = value; else if (field === 'clientCert') clientCertError = value; else clientKeyError = value;
  }
  async function readCertificate(event: Event, field: CertificateField): Promise<void> {
    const input = event.currentTarget as HTMLInputElement, file = input.files?.[0];
    if (file === undefined) return;
    setFieldError(field, '');
    if (file.size > maxCertificateBytes) {
      setCertificate(field, '');
      setFileName(field, '');
      setFieldError(field, 'Choose a file that is 16 KB or smaller.');
      input.value = '';
      return;
    }
    try {
      const text = await file.text();
      if (!text.includes('-----BEGIN')) {
        setFieldError(field, 'This file does not contain a PEM block.');
        setCertificate(field, '');
        setFileName(field, '');
        input.value = '';
        return;
      }
      setCertificate(field, text);
      setFileName(field, file.name);
    } catch {
      setFieldError(field, 'Could not read this file. Choose it again.');
      setCertificate(field, '');
      setFileName(field, '');
    }
  }
  function requiredCertificateError(isSaved: boolean, value: string, currentError: string, missingMessage: string): string {
    if (currentError !== '') return currentError;
    return isSaved || value !== '' ? '' : missingMessage;
  }
  function validateCertificates(): boolean {
    if (tlsChoice === 'none') return true;
    caError = requiredCertificateError(caSet, caCert, caError, 'Choose a CA certificate.');
    if (tlsChoice === 'full') {
      clientCertError = requiredCertificateError(clientCertSet, clientCert, clientCertError, 'Choose a client certificate.');
      clientKeyError = requiredCertificateError(clientKeySet, clientKey, clientKeyError, 'Choose a client key.');
    }
    return [caError, ...(tlsChoice === 'full' ? [clientCertError, clientKeyError] : [])].every((error) => error === '');
  }
  async function save(removal?: Removal): Promise<void> {
    if (removal === undefined && !validateCertificates()) return;
    saving = true;
    failed = false;
    message = '';
    try {
      const result = await saveSharedDatabaseSettings({
        host, port, database, user, password, sslMode: sslModeForTLSChoice(tlsChoice),
        caCert: tlsChoice === 'none' ? '' : caCert,
        clientCert: tlsChoice === 'full' ? clientCert : '',
        clientKey: tlsChoice === 'full' ? clientKey : '',
        removeCaCert: removal === 'ca',
        removeClientCert: removal === 'client'
      });
      password = ''; caCert = ''; clientCert = ''; clientKey = '';
      caFileName = ''; clientCertFileName = ''; clientKeyFileName = '';
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
    const removal = pendingRemoval; pendingRemoval = undefined;
    if (removal !== undefined) await save(removal);
  }

  function submit(event: SubmitEvent): void {
    event.preventDefault(); void save();
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
        <FormField id="database-host" label="Host">{#snippet children({ describedby, invalid })}<input id="database-host" class="input input-sm w-full" aria-describedby={describedby} aria-invalid={invalid} bind:value={host} required />{/snippet}</FormField>
        <FormField id="database-port" label="Port">{#snippet children({ describedby, invalid })}<input id="database-port" class="input input-sm w-full" type="number" min="1" max="65535" aria-describedby={describedby} aria-invalid={invalid} bind:value={port} required />{/snippet}</FormField>
        <FormField id="database-name" label="Database">{#snippet children({ describedby, invalid })}<input id="database-name" class="input input-sm w-full" aria-describedby={describedby} aria-invalid={invalid} bind:value={database} required />{/snippet}</FormField>
        <FormField id="database-user" label="User">{#snippet children({ describedby, invalid })}<input id="database-user" class="input input-sm w-full" aria-describedby={describedby} aria-invalid={invalid} bind:value={user} required />{/snippet}</FormField>
        <FormField id="database-password" label="Password" optional={passwordSet}>{#snippet children({ describedby, invalid })}<input id="database-password" class="input input-sm w-full" type="password" autocomplete="new-password" placeholder={passwordSet ? 'Saved; leave blank to keep it' : ''} aria-describedby={describedby} aria-invalid={invalid} bind:value={password} required={!passwordSet} />{/snippet}</FormField>
        <FormField id="database-tls" label="TLS" hint="Choose how the server connection is protected.">
          {#snippet children({ describedby, invalid })}<select id="database-tls" class="select select-sm w-full" bind:value={tlsChoice} aria-describedby={describedby} aria-invalid={invalid}><option value="none">No TLS</option><option value="ca">CA certificate only</option><option value="full">Full (CA + client certificate)</option></select>{/snippet}
        </FormField>
      </div>

      {#if tlsChoice !== 'none'}
        <FormField id="database-ca-cert" label="CA certificate" error={caError}>
          {#snippet children({ describedby, invalid })}
            <div class="space-y-2">
              {#if caSet}<div class="flex items-center gap-2"><span class="badge badge-sm badge-success badge-soft">Saved</span><button class="btn btn-ghost btn-xs btn-square" type="button" aria-label="Remove saved CA certificate" disabled={saving} onclick={() => (pendingRemoval = 'ca')}><Icon name="trash" class="size-4" /></button></div>{/if}
              <input id="database-ca-cert" class="file-input file-input-sm w-full" type="file" accept={certificateAccept} aria-describedby={describedby} aria-invalid={invalid} onchange={(event) => void readCertificate(event, 'ca')} />
              {#if caFileName !== ''}<p class="text-xs" aria-live="polite">{caFileName}</p>{/if}
            </div>
          {/snippet}
        </FormField>
      {/if}

      {#if tlsChoice === 'full'}
        <div class="grid gap-3 sm:grid-cols-2">
          <FormField id="database-client-cert" label="Client certificate" error={clientCertError}>
            {#snippet children({ describedby, invalid })}
              <div class="space-y-2">
                {#if clientCertSet || clientKeySet}<div class="flex items-center gap-2">{#if clientCertSet}<span class="badge badge-sm badge-success badge-soft">Saved</span>{/if}<button class="btn btn-ghost btn-xs btn-square" type="button" aria-label="Remove saved client certificate and key" disabled={saving} onclick={() => (pendingRemoval = 'client')}><Icon name="trash" class="size-4" /></button></div>{/if}
                <input id="database-client-cert" class="file-input file-input-sm w-full" type="file" accept={certificateAccept} aria-describedby={describedby} aria-invalid={invalid} onchange={(event) => void readCertificate(event, 'clientCert')} />
                {#if clientCertFileName !== ''}<p class="text-xs" aria-live="polite">{clientCertFileName}</p>{/if}
              </div>
            {/snippet}
          </FormField>
          <FormField id="database-client-key" label="Client key" error={clientKeyError}>
            {#snippet children({ describedby, invalid })}
              <div class="space-y-2">
                {#if clientKeySet}<span class="badge badge-sm badge-success badge-soft">Saved</span>{/if}
                <input id="database-client-key" class="file-input file-input-sm w-full" type="file" accept={certificateAccept} aria-describedby={describedby} aria-invalid={invalid} onchange={(event) => void readCertificate(event, 'clientKey')} />
                {#if clientKeyFileName !== ''}<p class="text-xs" aria-live="polite">{clientKeyFileName}</p>{/if}
              </div>
            {/snippet}
          </FormField>
        </div>
      {/if}

      <div class="flex flex-wrap items-center gap-3">
        <button class="btn btn-primary btn-sm" type="submit" disabled={saving}>{saving ? 'Saving…' : 'Save database settings'}</button>
        {#if message !== ''}<p class="text-xs {failed ? 'text-error' : 'text-[var(--brand-text-secondary)]'}" role={failed ? 'alert' : 'status'}>{message}</p>{/if}
      </div>
    </form>
  </SettingsSection>
{/if}

<ConfirmModal open={pendingRemoval !== undefined} title={pendingRemoval === 'ca' ? 'Remove saved CA certificate?' : 'Remove saved client certificate and key?'} body="This removes the saved certificate material from shared database settings." confirmLabel="Remove" cancelLabel="Keep it" oncancel={() => (pendingRemoval = undefined)} onconfirm={confirmRemoval} />
