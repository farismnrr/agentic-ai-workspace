<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '../../../shared/api/http';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import FormField from '../components/FormField.svelte';
  import DatabaseCertificates from './DatabaseCertificates.svelte';
  import { certificateErrors, databaseDefaultPorts, databaseKinds, databaseNameLabels, getDatabaseKinds, getDatabaseSettings, saveDatabaseSettings, tlsChoiceFromSettings, tlsForChoice, type DatabaseKind, type DatabaseSettings, type Removal, type TLSChoice } from './database-api';
  import SettingsSection from './parts/SettingsSection.svelte';

  let kind = $state<DatabaseKind>('postgres');
  let availability = $state<Record<DatabaseKind, boolean>>({ postgres: false, mysql: false, mongodb: false });
  let host = $state(''), database = $state(''), user = $state(''), password = $state('');
  let port = $state(5432), tlsChoice = $state<TLSChoice>('none');
  let caCert = $state(''), clientCert = $state(''), clientKey = $state('');
  let caFileName = $state(''), clientCertFileName = $state(''), clientKeyFileName = $state('');
  let caError = $state(''), clientCertError = $state(''), clientKeyError = $state('');
  let passwordSet = $state(false), caSet = $state(false), clientCertSet = $state(false), clientKeySet = $state(false);
  let loading = $state(true), loadFailed = $state(false), saving = $state(false), failed = $state(false);
  let message = $state(''), pendingRemoval = $state<Removal | undefined>();

  onMount(() => { void loadKinds(); });
  async function loadKinds(): Promise<void> {
    loading = true; loadFailed = false;
    try {
      const kinds = await getDatabaseKinds();
      const next = { postgres: false, mysql: false, mongodb: false };
      for (const item of kinds) next[item.kind] = item.configured;
      availability = next;
    } catch { loadFailed = true; message = 'Could not load database settings.'; loading = false; return; }
    await load();
  }
  async function load(): Promise<void> {
    loading = true; loadFailed = false; message = '';
    host = ''; database = ''; user = ''; password = ''; port = databaseDefaultPorts[kind];
    caCert = ''; clientCert = ''; clientKey = ''; caFileName = ''; clientCertFileName = ''; clientKeyFileName = '';
    caError = ''; clientCertError = ''; clientKeyError = '';
    try {
      const settings = await getDatabaseSettings(kind);
      host = settings.host; port = settings.port || databaseDefaultPorts[kind]; database = settings.database; user = settings.user;
      tlsChoice = settings.configured ? tlsChoiceFromSettings(settings.tls, settings.clientCertSet) : 'none';
      passwordSet = settings.passwordSet; caSet = settings.caSet; clientCertSet = settings.clientCertSet; clientKeySet = settings.clientKeySet;
      availability = { ...availability, [kind]: settings.configured };
    } catch { loadFailed = true; message = 'Could not load database settings.'; }
    finally { loading = false; }
  }
  function switchKind(next: DatabaseKind): void { if (next !== kind) { kind = next; void load(); } }
  function setCertificate(field: 'ca' | 'clientCert' | 'clientKey', value: string, fileName: string): void {
    if (field === 'ca') { caCert = value; caFileName = fileName; }
    else if (field === 'clientCert') { clientCert = value; clientCertFileName = fileName; }
    else { clientKey = value; clientKeyFileName = fileName; }
  }
  function errorMessage(error: unknown): string {
    if (error instanceof ApiError && error.code === 'invalid_settings') return 'Check the connection details and certificate files, then try again.';
    return 'Could not save these settings. Check the fields and try again.';
  }
  function validateCertificates(): boolean {
    const errors = certificateErrors(tlsChoice, { caSet, clientCertSet, clientKeySet, caCert, clientCert, clientKey, caError, clientCertError, clientKeyError });
    caError = errors.caError; clientCertError = errors.clientCertError; clientKeyError = errors.clientKeyError;
    return Object.values(errors).every((error) => error === '');
  }
  async function save(removal?: Removal): Promise<void> {
    if (removal === undefined && !validateCertificates()) return;
    saving = true; failed = false; message = '';
    try {
      const result: DatabaseSettings = await saveDatabaseSettings(kind, {
        host, port, database, user, password, tls: tlsForChoice(tlsChoice),
        caCert: tlsChoice === 'none' ? '' : caCert,
        clientCert: tlsChoice === 'full' ? clientCert : '', clientKey: tlsChoice === 'full' ? clientKey : '',
        removeCaCert: removal === 'ca', removeClientCert: removal === 'client',
      });
      password = ''; caCert = ''; clientCert = ''; clientKey = ''; caFileName = ''; clientCertFileName = ''; clientKeyFileName = '';
      caError = ''; clientCertError = ''; clientKeyError = '';
      passwordSet = result.passwordSet; caSet = result.caSet; clientCertSet = result.clientCertSet; clientKeySet = result.clientKeySet;
      availability = { ...availability, [kind]: result.configured };
      message = removal === 'ca' ? 'CA certificate removed.' : removal === 'client' ? 'Client certificate and key removed.' : 'Database settings saved.';
    } catch (error) { failed = true; message = errorMessage(error); }
    finally { saving = false; }
  }
  async function confirmRemoval(): Promise<void> { const removal = pendingRemoval; pendingRemoval = undefined; if (removal !== undefined) await save(removal); }
  function requestRemoval(removal: Removal): void { pendingRemoval = removal; }
  function submit(event: SubmitEvent): void { event.preventDefault(); void save(); }
</script>

{#if loading}
  <div class="py-3" role="status"><span class="loading loading-spinner loading-sm" aria-label="Loading database settings"></span></div>
{:else if loadFailed}
  <div class="alert alert-error alert-soft" role="alert"><span>{message}</span><button class="btn btn-sm" type="button" onclick={() => { void loadKinds(); }}>Try again</button></div>
{:else}
  <SettingsSection title="Database" description="Set databases available to projects.">
    <div class="space-y-4 py-3">
      <div role="tablist" class="tabs tabs-border w-full flex-nowrap" aria-label="Database kind">
        {#each databaseKinds as item (item)}
          <button type="button" role="tab" class="tab min-w-0 flex-1 gap-1 px-1 text-xs sm:gap-2 sm:text-sm" class:tab-active={kind === item} aria-selected={kind === item} onclick={() => { switchKind(item); }}>
            <span>{item === 'postgres' ? 'PostgreSQL' : item === 'mysql' ? 'MySQL' : 'MongoDB'}</span><span class="badge badge-xs whitespace-nowrap" class:badge-success={availability[item]}>{availability[item] ? 'Set up' : 'Not set up'}</span>
          </button>
        {/each}
      </div>
      <form class="space-y-4" onsubmit={submit}>
        <div class="grid gap-3 sm:grid-cols-2">
          <FormField id="database-host" label="Host">{#snippet children({ describedby, invalid })}<input id="database-host" class="input input-sm w-full" aria-describedby={describedby} aria-invalid={invalid} bind:value={host} required />{/snippet}</FormField>
          <FormField id="database-port" label="Port">{#snippet children({ describedby, invalid })}<input id="database-port" class="input input-sm w-full" type="number" min="1" max="65535" aria-describedby={describedby} aria-invalid={invalid} bind:value={port} required />{/snippet}</FormField>
          <FormField id="database-name" label={databaseNameLabels[kind]} hint={kind === 'mongodb' ? 'MongoDB checks this database when signing in; it is usually admin.' : undefined}>{#snippet children({ describedby, invalid })}<input id="database-name" class="input input-sm w-full" aria-describedby={describedby} aria-invalid={invalid} bind:value={database} required />{/snippet}</FormField>
          <FormField id="database-user" label="Admin user">{#snippet children({ describedby, invalid })}<input id="database-user" class="input input-sm w-full" aria-describedby={describedby} aria-invalid={invalid} bind:value={user} required />{/snippet}</FormField>
          <FormField id="database-password" label="Password" optional={passwordSet}>{#snippet children({ describedby, invalid })}<input id="database-password" class="input input-sm w-full" type="password" autocomplete="new-password" placeholder={passwordSet ? 'Saved; leave blank to keep it' : ''} aria-describedby={describedby} aria-invalid={invalid} bind:value={password} required={!passwordSet} />{/snippet}</FormField>
          <FormField id="database-tls" label="TLS" hint="Choose the connection's TLS protection.">{#snippet children({ describedby, invalid })}<select id="database-tls" class="select select-sm w-full" bind:value={tlsChoice} aria-describedby={describedby} aria-invalid={invalid}><option value="none">No TLS</option><option value="ca">CA only</option><option value="full">Full (CA + client certificate)</option></select>{/snippet}</FormField>
        </div>
        <DatabaseCertificates bind:caFileName bind:clientCertFileName bind:clientKeyFileName bind:caError bind:clientCertError bind:clientKeyError {tlsChoice} {caSet} {clientCertSet} {clientKeySet} {saving} onfile={setCertificate} onremove={requestRemoval} />
        <div class="flex flex-wrap items-center gap-3">
          <button class="btn btn-primary btn-sm" type="submit" disabled={saving}>{saving ? 'Saving…' : 'Save database settings'}</button>
          {#if message !== ''}<p class="text-xs" class:text-error={failed} role={failed ? 'alert' : 'status'}>{message}</p>{/if}
        </div>
      </form>
    </div>
  </SettingsSection>
{/if}

<ConfirmModal open={pendingRemoval !== undefined} title={pendingRemoval === 'ca' ? 'Remove saved CA certificate?' : 'Remove saved client certificate and key?'} body="This removes the saved certificate material from database settings." confirmLabel="Remove" cancelLabel="Keep it" oncancel={() => (pendingRemoval = undefined)} onconfirm={confirmRemoval} />
