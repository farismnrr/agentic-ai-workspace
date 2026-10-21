<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '../../../../shared/api/http';
  import ConfirmModal from '../../components/ConfirmModal.svelte';
  import SettingsSection from '../../settings/parts/SettingsSection.svelte';
  import { detachSQLiteDatabase, getDatabase, provisionSharedDatabase, testDatabase, type DatabaseConnection, type TestResult } from './env-api';
  import { databaseTestErrorMessage, sharedProvisionErrorMessage } from './env-logic';

  interface Props { projectId: string; onenvchange: () => void }
  const { projectId, onenvchange }: Props = $props();
  let saved = $state<DatabaseConnection>();
  let choice = $state<'sqlite' | 'shared'>('sqlite');
  let loading = $state(true);
  let loadFailed = $state(false);
  let busy = $state(false);
  let testing = $state(false);
  let result = $state<TestResult>();
  let message = $state('');
  let messageError = $state(false);
  let confirmChoice = $state<'sqlite' | 'shared'>();

  const legacy = $derived(saved?.configured === true && !saved.managed);
  const sharedConfigured = $derived(saved?.sharedPostgres === true);

  onMount(() => { void load(); });

  async function load(): Promise<void> {
    try {
      saved = await getDatabase(projectId);
      choice = saved.configured && saved.managed ? 'shared' : 'sqlite';
    } catch {
      loadFailed = true;
    } finally {
      loading = false;
    }
  }

  function chooseMode(next: 'sqlite' | 'shared'): void {
    message = '';
    result = undefined;
    if (legacy || (next === 'sqlite' && saved?.configured)) {
      confirmChoice = next;
      return;
    }
    if (next === 'shared') {
      choice = next;
      void createSharedSchema();
      return;
    }
    choice = next;
  }

  async function createSharedSchema(): Promise<void> {
    if (!sharedConfigured) return;
    busy = true;
    message = 'Preparing the shared database…';
    messageError = false;
    try {
      const connection = await provisionSharedDatabase(projectId);
      saved = { ...connection, sharedPostgres: saved?.sharedPostgres ?? false };
      choice = 'shared';
      message = 'The shared database is ready for this project.';
      onenvchange();
    } catch (cause) {
      const code = cause instanceof ApiError ? cause.code : undefined;
      message = sharedProvisionErrorMessage(code);
      messageError = true;
    } finally {
      busy = false;
    }
  }

  async function checkConnection(): Promise<void> {
    testing = true;
    result = undefined;
    try {
      result = await testDatabase(projectId);
    } catch (cause) {
      const code = cause instanceof ApiError ? cause.code : undefined;
      result = { ok: false, message: databaseTestErrorMessage(code) };
    } finally {
      testing = false;
    }
  }

  async function confirmChange(): Promise<void> {
    const next = confirmChoice;
    if (next === undefined) return;
    confirmChoice = undefined;
    busy = true;
    try {
      await detachSQLiteDatabase(projectId);
      if (saved !== undefined) saved = { ...saved, configured: false, managed: false };
      onenvchange();
      if (next === 'shared') {
        choice = 'sqlite';
        await createSharedSchema();
      } else {
        choice = 'sqlite';
        message = 'SQLite selected. The existing database and its data are untouched.';
        messageError = false;
      }
    } catch {
      message = 'Could not change this project’s database choice.';
      messageError = true;
    } finally {
      busy = false;
    }
  }
</script>

<SettingsSection title="Database" description="Choose where this project stores its data.">
  {#if loading}
    <div class="py-3" role="status"><span class="loading loading-spinner loading-sm" aria-label="Loading database settings"></span></div>
  {:else if loadFailed}
    <div class="py-3 alert alert-error alert-soft" role="alert">Could not load database settings.</div>
  {:else if legacy}
    <div class="space-y-3 py-4">
      <div class="alert alert-warning alert-soft">
        <div>
          <p class="font-medium">Custom (legacy)</p>
          <p class="text-sm">This saved connection is read-only. Switching does not change the external database or its data.</p>
        </div>
      </div>
      <dl class="grid gap-2 text-sm sm:grid-cols-2">
        <div><dt class="text-[var(--brand-text-secondary)]">Type</dt><dd>{saved?.type === 'postgres' ? 'PostgreSQL' : saved?.type === 'mysql' ? 'MySQL' : 'MongoDB'}</dd></div>
        <div><dt class="text-[var(--brand-text-secondary)]">Host and port</dt><dd class="break-all">{saved?.host}:{saved?.port}</dd></div>
        <div><dt class="text-[var(--brand-text-secondary)]">Database</dt><dd class="break-all">{saved?.database}</dd></div>
        <div><dt class="text-[var(--brand-text-secondary)]">User</dt><dd class="break-all">{saved?.user}</dd></div>
      </dl>
      <div class="flex flex-wrap gap-2">
        <button class="btn btn-ghost btn-sm" type="button" onclick={() => { chooseMode('sqlite'); }} disabled={busy}>Switch to SQLite</button>
        <button class="btn btn-primary btn-sm" type="button" onclick={() => { chooseMode('shared'); }} disabled={busy || !sharedConfigured}>Use shared database</button>
      </div>
      {#if !sharedConfigured}<p id="shared-database-hint" class="text-xs text-[var(--brand-text-secondary)]">An admin can configure this in Settings → Database.</p>{/if}
    </div>
  {:else}
    <div class="space-y-4 py-4">
      <fieldset class="fieldset grid gap-2 sm:grid-cols-2">
        <legend class="fieldset-legend">Project database</legend>
        <label class="flex cursor-pointer items-center gap-2">
          <input class="radio radio-sm" type="radio" name="database-mode" value="sqlite" checked={choice === 'sqlite'} onchange={() => { chooseMode('sqlite'); }} disabled={busy} />
          <span>SQLite (default)</span>
        </label>
        <label class="flex items-center gap-2" class:cursor-pointer={sharedConfigured}>
          <input class="radio radio-sm" type="radio" name="database-mode" value="shared" checked={choice === 'shared'} onchange={() => { chooseMode('shared'); }} disabled={!sharedConfigured || busy} aria-describedby={!sharedConfigured ? 'shared-database-hint' : undefined} />
          <span>Shared database</span>
        </label>
      </fieldset>
      {#if !sharedConfigured}<p id="shared-database-hint" class="text-xs text-[var(--brand-text-secondary)]">An admin can configure this in Settings → Database.</p>{/if}
      {#if choice === 'shared' && saved?.configured && saved.managed}
        <div class="space-y-3 py-3">
          <p class="font-medium">Shared PostgreSQL</p>
          <dl class="grid gap-2 text-sm sm:grid-cols-3">
            <div><dt class="text-[var(--brand-text-secondary)]">Host</dt><dd class="break-all">{saved.host}</dd></div>
            <div><dt class="text-[var(--brand-text-secondary)]">Database</dt><dd class="break-all">{saved.database}</dd></div>
            <div><dt class="text-[var(--brand-text-secondary)]">User</dt><dd class="break-all">{saved.user}</dd></div>
          </dl>
          <p class="text-sm text-[var(--brand-text-secondary)]">Switching to SQLite keeps this schema and its data in the shared database.</p>
          <div class="flex flex-wrap gap-2">
            <button class="btn btn-outline btn-sm" type="button" onclick={() => void checkConnection()} disabled={testing || busy} aria-busy={testing}>
              {#if testing}<span class="loading loading-spinner loading-xs" aria-hidden="true"></span>{/if}Test connection
            </button>
            <button class="btn btn-ghost btn-sm" type="button" onclick={() => { chooseMode('sqlite'); }} disabled={busy}>Switch to SQLite</button>
          </div>
        </div>
      {:else if choice === 'shared'}
        <div class="space-y-3">
          <p class="text-sm text-[var(--brand-text-secondary)]">A private schema and login will be created for this project.</p>
          <button class="btn btn-primary btn-sm" type="button" onclick={() => void createSharedSchema()} disabled={busy || !sharedConfigured} aria-busy={busy}>
            {#if busy}<span class="loading loading-spinner loading-xs" aria-hidden="true"></span>{/if}Create shared database
          </button>
        </div>
      {:else}
        <p class="text-sm text-[var(--brand-text-secondary)]">This project uses its own SQLite file.</p>
      {/if}
    </div>
  {/if}
  <div class="space-y-2 pb-3">
    {#if result !== undefined}<div class="alert alert-soft {result.ok ? 'alert-success' : 'alert-error'}" role={result.ok ? 'status' : 'alert'}>{result.message}</div>{/if}
    {#if message !== ''}<p class="text-xs" class:text-error={messageError} role={messageError ? 'alert' : 'status'}>{message}</p>{/if}
  </div>
</SettingsSection>

<ConfirmModal
  open={confirmChoice !== undefined}
  title={confirmChoice === 'sqlite' ? 'Switch to SQLite?' : 'Use the shared database?'}
  body={confirmChoice === 'shared' ? 'The external database and its data will be left untouched.' : 'The current database connection will be removed from this project. Its data stays in the external database.'}
  confirmLabel={confirmChoice === 'sqlite' ? 'Use SQLite' : 'Use shared database'}
  cancelLabel="Cancel"
  oncancel={() => (confirmChoice = undefined)}
  onconfirm={confirmChange}
/>
