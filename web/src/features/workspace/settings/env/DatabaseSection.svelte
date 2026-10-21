<script lang="ts">
  import { onMount } from 'svelte';
  import Icon from '../../../../shared/ui/Icon.svelte';
  import ConfirmModal from '../../components/ConfirmModal.svelte';
  import SettingsSection from '../parts/SettingsSection.svelte';
  import DatabaseActions from './DatabaseActions.svelte';
  import DatabaseFields from './DatabaseFields.svelte';
  import DatabaseReachabilityNote from './DatabaseReachabilityNote.svelte';
  import DatabaseTestResult from './DatabaseTestResult.svelte';
  import { ApiError } from '../../../../shared/api/http';
  import { deleteDatabase, getDatabase, provisionSharedDatabase, saveDatabase, testDatabase, type DatabaseConnection, type DatabaseInput, type DatabaseType, type TestResult } from '../env-api';
  import { databaseDefaults, databaseTestErrorMessage, sharedProvisionErrorMessage, type DatabaseMode } from '../env-logic';

  interface Props { projectId: string; onenvchange: () => void }
  const { projectId, onenvchange }: Props = $props();
  let saved = $state<DatabaseConnection | undefined>();
  let type = $state<DatabaseType>('postgres');
  let host = $state(''); let port = $state(5432); let database = $state('');
  let user = $state(''); let password = $state(''); let tls = $state(true);
  let envName = $state('DATABASE_URL');
  let result = $state<TestResult | undefined>();
  let message = $state(''); let messageError = $state(false); let busy = $state(false); let testing = $state(false);
  let confirmDelete = $state(false);
  let mode = $state<DatabaseMode>('manual');
  let sharedPostgres = $state(false);

  onMount(() => { void load(); });

  async function load(): Promise<void> {
    try {
      saved = await getDatabase(projectId);
      sharedPostgres = saved.sharedPostgres === true;
      if (saved.configured) {
        type = saved.type; host = saved.host; port = saved.port; database = saved.database;
        user = saved.user; tls = saved.tls; envName = saved.envName;
        if (saved.managed) mode = 'shared';
      }
    } catch { message = 'Could not load database settings.'; messageError = true; }
  }

  function chooseType(next: DatabaseType): void { type = next; port = databaseDefaults[next].port; envName = databaseDefaults[next].envName; }
  function chooseMode(next: DatabaseMode): void { mode = next; message = ''; messageError = false; result = undefined; }
  function input(): DatabaseInput { return { type, host: host.trim(), port, database: database.trim(), user: user.trim(), password, tls, envName: envName.trim() }; }
  function matchesSaved(): boolean {
    return saved?.configured === true && type === saved.type && host === saved.host && port === saved.port &&
      database === saved.database && user === saved.user && tls === saved.tls && envName === saved.envName;
  }
  function validate(requirePassword: boolean): boolean {
    if (!host.trim() || !database.trim() || !user.trim() || !/^[A-Z_][A-Z0-9_]{0,63}$/.test(envName.trim()) || !Number.isInteger(port) || port < 1 || port > 65535) {
      message = 'Complete each field with a valid name and port before continuing.'; messageError = true; return false;
    }
    if (requirePassword && password === '') { message = 'Enter the database password to save or test this connection.'; messageError = true; return false; }
    message = ''; messageError = false; return true;
  }
  async function test(): Promise<void> {
    if (!validate(password === '' && !matchesSaved())) return;
    busy = true; testing = true; result = undefined;
    try { result = await testDatabase(projectId, password === '' && matchesSaved() ? undefined : input()); }
    catch (cause) {
      const code = cause instanceof ApiError ? cause.code : undefined;
      result = { ok: false, message: databaseTestErrorMessage(code) };
    }
    finally { busy = false; testing = false; }
  }
  async function save(): Promise<void> {
    if (!validate(true)) return;
    busy = true;
    try {
      saved = await saveDatabase(projectId, input()); password = '';
      message = 'Database connection saved.'; messageError = false; result = undefined; onenvchange();
    } catch { message = 'Could not save this connection. Check the fields and try again.'; messageError = true; }
    finally { busy = false; }
  }
  async function remove(): Promise<void> {
    try {
      await deleteDatabase(projectId); saved = undefined; host = ''; database = ''; user = ''; password = '';
      message = 'Database connection deleted.'; messageError = false; onenvchange();
    } catch { message = 'Could not delete this database connection.'; messageError = true; }
    finally { confirmDelete = false; }
  }

  async function createSharedSchema(): Promise<void> {
    busy = true; result = undefined; message = 'Creating your schema…'; messageError = false;
    try {
      saved = await provisionSharedDatabase(projectId);
      mode = 'shared';
      message = 'Your shared Postgres schema is ready. Connection details are saved for this project.';
      messageError = false;
      onenvchange();
    } catch (cause) {
      const code = cause instanceof ApiError ? cause.code : undefined;
      message = sharedProvisionErrorMessage(code);
      messageError = true;
    } finally { busy = false; }
  }
</script>

<SettingsSection title="Database" description="Choose how this project connects to a database.">
  <div class="space-y-4 py-4">
    <DatabaseReachabilityNote />
    <p class="text-sm text-[var(--brand-text-secondary)]">With no database selected, projects keep using their own SQLite file.</p>
    {#if sharedPostgres}
      <div class="grid gap-2 sm:grid-cols-2" role="group" aria-label="Database connection">
        <label class="flex min-h-11 cursor-pointer items-center gap-2 rounded-lg border border-base-300 px-3 has-[:checked]:border-primary">
          <input class="radio radio-sm" type="radio" name="database-mode" value="shared" checked={mode === 'shared'} onchange={() => { chooseMode('shared'); }} />
          <span>Use shared Postgres (schema created automatically)</span>
        </label>
        <label class="flex min-h-11 cursor-pointer items-center gap-2 rounded-lg border border-base-300 px-3 has-[:checked]:border-primary">
          <input class="radio radio-sm" type="radio" name="database-mode" value="manual" checked={mode === 'manual'} onchange={() => { chooseMode('manual'); }} />
          <span>Manual connection</span>
        </label>
      </div>
    {/if}
    {#if saved?.configured && saved.managed && (!sharedPostgres || mode === 'manual')}
      <div class="space-y-3">
        <p class="rounded-lg border border-base-300 p-3 text-sm text-[var(--brand-text-secondary)]">
          {#if !sharedPostgres}
            This project uses a shared Postgres schema, but the shared admin is not configured on the server. Delete the connection to switch to manual.
          {:else}
            Delete this shared connection before setting up a manual connection. Its schema and data will be kept.
          {/if}
        </p>
        <DatabaseActions {busy} {testing} configured shared={true} ontest={() => void test()} onsave={() => void save()} ondelete={() => (confirmDelete = true)} />
      </div>
    {:else if mode === 'shared' && sharedPostgres}
      {#if saved?.configured && saved.managed}
        <div class="space-y-3 rounded-lg border border-base-300 p-3">
          <p class="font-medium">Managed Postgres connection</p>
          <dl class="grid gap-2 text-sm sm:grid-cols-3">
            <div><dt class="text-[var(--brand-text-secondary)]">Host</dt><dd class="break-all">{saved.host}</dd></div>
            <div><dt class="text-[var(--brand-text-secondary)]">Database</dt><dd class="break-all">{saved.database}</dd></div>
            <div><dt class="text-[var(--brand-text-secondary)]">User</dt><dd class="break-all">{saved.user}</dd></div>
          </dl>
          <p class="text-sm text-[var(--brand-text-secondary)]">Removing this connection keeps its schema and data. You can reconnect later.</p>
          <button class="btn btn-outline btn-sm min-h-11" type="button" onclick={() => void createSharedSchema()} disabled={busy} aria-busy={busy}>
            {#if busy}<span class="loading loading-spinner loading-xs" aria-hidden="true"></span>{/if}
            Reconnect and reset password
          </button>
          <DatabaseActions {busy} {testing} configured shared={true} ontest={() => void test()} onsave={() => void save()} ondelete={() => (confirmDelete = true)} />
        </div>
      {:else}
        <div class="space-y-3">
          <p class="text-sm text-[var(--brand-text-secondary)]">A private schema and login will be created for this project.</p>
          <button class="btn btn-primary btn-sm min-h-11" type="button" onclick={() => void createSharedSchema()} disabled={busy} aria-busy={busy}>
            {#if busy}<span class="loading loading-spinner loading-xs" aria-hidden="true"></span>{/if}
            Create my schema
          </button>
        </div>
      {/if}
    {:else}
      {#if saved?.configured}<p class="text-sm text-[var(--brand-text-secondary)]">Connection saved. Password saved and hidden.</p>{/if}
      <DatabaseFields {type} {host} {port} {database} {user} {password} {tls} {envName} configured={saved?.configured === true}
        ontype={chooseType} onhost={(value: string) => { host = value; }} onport={(value: number) => { port = value; }}
        ondatabase={(value: string) => { database = value; }} onuser={(value: string) => { user = value; }}
        onpassword={(value: string) => { password = value; }} ontls={(value: boolean) => { tls = value; }} onenvname={(value: string) => { envName = value; }} />
      <DatabaseActions {busy} {testing} configured={saved?.configured === true} ontest={() => void test()} onsave={() => void save()} ondelete={() => (confirmDelete = true)} />
    {/if}
    <div class="min-h-6 space-y-2" aria-live="polite">
      <DatabaseTestResult {result} />
      {#if message !== ''}<p class="flex items-center gap-2 text-sm" class:text-error={messageError}><Icon name={messageError ? 'alert-triangle' : 'check-circle'} class="size-4 shrink-0" />{message}</p>{/if}
    </div>
  </div>
</SettingsSection>

<ConfirmModal open={confirmDelete} title="Delete this database connection?"
  body={saved?.managed ? "Its environment variable will be removed. The schema and its data will be kept." : "Its environment variable will also be removed from this project."}
  confirmLabel="Delete connection" cancelLabel="Cancel"
  oncancel={() => (confirmDelete = false)} onconfirm={remove} />
