<script lang="ts">
  import { onMount } from 'svelte';
  import Icon from '../../../../shared/ui/Icon.svelte';
  import ConfirmModal from '../../components/ConfirmModal.svelte';
  import SettingsSection from '../parts/SettingsSection.svelte';
  import DatabaseActions from './DatabaseActions.svelte';
  import DatabaseFields from './DatabaseFields.svelte';
  import DatabaseReachabilityNote from './DatabaseReachabilityNote.svelte';
  import DatabaseTestResult from './DatabaseTestResult.svelte';
  import { deleteDatabase, getDatabase, saveDatabase, testDatabase, type DatabaseConnection, type DatabaseInput, type DatabaseType, type TestResult } from '../env-api';
  import { databaseDefaults } from '../env-logic';

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

  onMount(() => { void load(); });

  async function load(): Promise<void> {
    try {
      saved = await getDatabase(projectId);
      if (saved.configured) {
        type = saved.type; host = saved.host; port = saved.port; database = saved.database;
        user = saved.user; tls = saved.tls; envName = saved.envName;
      }
    } catch { message = 'Could not load database settings.'; messageError = true; }
  }

  function chooseType(next: DatabaseType): void { type = next; port = databaseDefaults[next].port; envName = databaseDefaults[next].envName; }
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
    catch { result = { ok: false, message: 'Could not test this connection. Check the fields and try again.' }; }
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
</script>

<SettingsSection title="Database" description="Store a connection address for project runs.">
  <div class="space-y-4 py-4">
    <DatabaseReachabilityNote />
    {#if saved?.configured}<p class="text-sm text-[var(--brand-text-secondary)]">Connection saved. Password saved and hidden.</p>{/if}
    <DatabaseFields {type} {host} {port} {database} {user} {password} {tls} {envName} configured={saved?.configured === true}
      ontype={chooseType} onhost={(value: string) => { host = value; }} onport={(value: number) => { port = value; }}
      ondatabase={(value: string) => { database = value; }} onuser={(value: string) => { user = value; }}
      onpassword={(value: string) => { password = value; }} ontls={(value: boolean) => { tls = value; }} onenvname={(value: string) => { envName = value; }} />
    <DatabaseActions {busy} {testing} configured={saved?.configured === true} ontest={() => void test()} onsave={() => void save()} ondelete={() => (confirmDelete = true)} />
    <div class="min-h-6 space-y-2" aria-live="polite">
      <DatabaseTestResult {result} />
      {#if message !== ''}<p class="flex items-center gap-2 text-sm" class:text-error={messageError}><Icon name={messageError ? 'alert-triangle' : 'check-circle'} class="size-4 shrink-0" />{message}</p>{/if}
    </div>
  </div>
</SettingsSection>

<ConfirmModal open={confirmDelete} title="Delete this database connection?"
  body="Its environment variable will also be removed from this project."
  confirmLabel="Delete connection" cancelLabel="Cancel"
  oncancel={() => (confirmDelete = false)} onconfirm={remove} />
