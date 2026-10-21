<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '../../../../shared/api/http';
  import ConfirmModal from '../../components/ConfirmModal.svelte';
  import FormField from '../../components/FormField.svelte';
  import SettingsSection from '../../settings/parts/SettingsSection.svelte';
  import { detachSQLiteDatabase, getDatabase, getDatabaseKinds, provisionDatabase, testDatabase, type DatabaseConnection, type DatabaseKind, type DatabaseKindAvailability, type TestResult } from './env-api';
  import { databaseProvisionErrorMessage, databaseTestErrorMessage } from './env-logic';

  interface Props { projectId: string; onenvchange: () => void }
  const { projectId, onenvchange }: Props = $props();
  type Choice = DatabaseConnection['choice'];
  let saved = $state<DatabaseConnection>();
  let kinds = $state<DatabaseKindAvailability[]>([]);
  let choice = $state<Choice>('sqlite');
  let loading = $state(true), loadFailed = $state(false), busy = $state(false), testing = $state(false);
  let result = $state<TestResult>(), message = $state(''), messageError = $state(false);
  let confirmChoice = $state<DatabaseKind | 'sqlite'>();

  const legacy = $derived(choice === 'custom');
  const isAdmin = $derived(kinds.some((item) => item.admin));
  const notSetUpKinds = $derived(kinds.filter((item) => !item.configured).map((item) => kindLabels[item.kind]));
  const showSetupHint = $derived(notSetUpKinds.length > 0);
  const setupHint = $derived(isAdmin
    ? `${notSetUpKinds.join(', ')}${notSetUpKinds.length === 1 ? " isn't" : " aren't"} set up yet - add ${notSetUpKinds.length === 1 ? 'it' : 'them'} in`
    : "Some databases aren't set up yet - ask an admin to add them in Settings.");
  const availableKinds: DatabaseKind[] = ['postgres', 'mysql', 'mongodb'];
  const kindLabels: Record<DatabaseKind, string> = { postgres: 'PostgreSQL', mysql: 'MySQL', mongodb: 'MongoDB' };
  onMount(() => { void load(); });

  async function load(): Promise<void> {
    loading = true; loadFailed = false;
    try {
      const [connection, available] = await Promise.all([getDatabase(projectId), getDatabaseKinds()]);
      saved = connection; kinds = available; choice = connection.choice;
    } catch { loadFailed = true; }
    finally { loading = false; }
  }
  function configured(kind: DatabaseKind): boolean { return kinds.find((item) => item.kind === kind)?.configured ?? false; }
  function databaseLabel(kind: DatabaseConnection['type']): string { return kind === '' ? 'Custom' : kindLabels[kind]; }
  function selectChoice(value: string): void {
    if (value === choice) return;
    message = ''; result = undefined;
    if (value === 'sqlite') { selectSQLite(); return; }
    const target = availableKinds.find((kind) => kind === value);
    if (target === undefined || !configured(target)) return;
    if (needsConfirmation(target)) confirmChoice = target;
    else void provision(target);
  }
  function selectSQLite(): void { if (saved?.configured) confirmChoice = 'sqlite'; else choice = 'sqlite'; }
  function needsConfirmation(target: DatabaseKind): boolean { return saved?.configured === true && (legacy || saved.type !== target); }
  async function provision(kind: DatabaseKind, detachLegacy = false): Promise<void> {
    busy = true; message = 'Preparing the database for this project…'; messageError = false;
    try {
      if (detachLegacy) {
        await detachSQLiteDatabase(projectId);
        saved = undefined; choice = 'sqlite'; onenvchange();
      }
      saved = await provisionDatabase(projectId, kind);
      choice = kind; message = `${kindLabels[kind]} is ready for this project.`; onenvchange();
    } catch (cause) {
      const code = cause instanceof ApiError ? cause.code : undefined;
      message = databaseProvisionErrorMessage(code); messageError = true;
    } finally { busy = false; }
  }
  async function detach(): Promise<void> {
    busy = true; message = '';
    try {
      await detachSQLiteDatabase(projectId);
      saved = undefined; choice = 'sqlite'; onenvchange();
      message = 'SQLite selected. Data in the database server is kept.'; messageError = false;
    } catch { message = 'Could not change this project’s database choice.'; messageError = true; }
    finally { busy = false; }
  }
  async function confirmChange(): Promise<void> {
    const target = confirmChoice; confirmChoice = undefined;
    if (target === undefined) return;
    if (target === 'sqlite') { await detach(); return; }
    await provision(target, legacy);
  }
  function confirmationTitle(): string {
    if (confirmChoice === 'sqlite') return 'Switch to SQLite?';
    return confirmChoice === undefined ? 'Change database?' : `Use ${kindLabels[confirmChoice]}?`;
  }
  async function checkConnection(): Promise<void> {
    testing = true; result = undefined;
    try { result = await testDatabase(projectId); }
    catch (cause) { result = { ok: false, message: databaseTestErrorMessage(cause instanceof ApiError ? cause.code : undefined) }; }
    finally { testing = false; }
  }
</script>

<SettingsSection title="Database" description="Choose where this project stores its data.">
  {#if loading}
    <div class="py-3" role="status"><span class="loading loading-spinner loading-sm" aria-label="Loading database settings"></span></div>
  {:else if loadFailed}
    <div class="py-3 alert alert-error alert-soft" role="alert"><span>Could not load database settings.</span><button class="btn btn-sm" type="button" onclick={() => { void load(); }}>Try again</button></div>
  {:else}
    <div class="space-y-4 py-4">
      <FormField id="project-database" label="Database" hint={showSetupHint ? setupHint : undefined} hintLink={showSetupHint && isAdmin ? { href: '/app/settings/database', label: 'Settings → Database' } : undefined}>
        {#snippet children({ describedby, invalid })}
          <select id="project-database" class="select select-sm w-full" value={choice} aria-describedby={describedby} aria-invalid={invalid} onchange={(event) => { selectChoice(event.currentTarget.value); }} disabled={busy}>
            <option value="sqlite">SQLite (default)</option>
            {#each availableKinds as kind (kind)}<option value={kind} disabled={!configured(kind)}>{kindLabels[kind]}{configured(kind) ? '' : ' - not set up yet'}</option>{/each}
            {#if legacy}<option value="custom" disabled>Custom (legacy)</option>{/if}
          </select>
        {/snippet}
      </FormField>
      {#if legacy}
        <div class="alert alert-warning alert-soft">
          <div>
            <p class="font-medium">Custom (legacy)</p>
            <p class="text-sm">This saved connection is read-only. Switching keeps the external database and its data untouched.</p>
          </div>
        </div>
        <dl class="grid gap-2 text-sm sm:grid-cols-2">
          <div><dt class="text-[var(--brand-text-secondary)]">Type</dt><dd>{saved === undefined ? 'Custom' : databaseLabel(saved.type)}</dd></div>
          <div><dt class="text-[var(--brand-text-secondary)]">Host and port</dt><dd class="break-all">{saved?.host}:{saved?.port}</dd></div>
          <div><dt class="text-[var(--brand-text-secondary)]">Database</dt><dd class="break-all">{saved?.database}</dd></div>
          <div><dt class="text-[var(--brand-text-secondary)]">User</dt><dd class="break-all">{saved?.user}</dd></div>
        </dl>
      {:else if saved?.configured && saved.managed}
        <div class="space-y-3 py-3">
          <p class="font-medium">{databaseLabel(saved.type)} for this project</p>
          <dl class="grid gap-2 text-sm sm:grid-cols-3">
            <div><dt class="text-[var(--brand-text-secondary)]">Host</dt><dd class="break-all">{saved.host}</dd></div>
            <div><dt class="text-[var(--brand-text-secondary)]">Database</dt><dd class="break-all">{saved.database}</dd></div>
            <div><dt class="text-[var(--brand-text-secondary)]">User</dt><dd class="break-all">{saved.user}</dd></div>
          </dl>
          <p class="text-sm text-[var(--brand-text-secondary)]">Switching to SQLite keeps this database and its data on the server.</p>
          <div class="flex flex-wrap gap-2">
            <button class="btn btn-outline btn-sm" type="button" onclick={() => void checkConnection()} disabled={testing || busy} aria-busy={testing}>{#if testing}<span class="loading loading-spinner loading-xs" aria-hidden="true"></span>{/if}Test connection</button>
          </div>
        </div>
      {:else}<p class="text-sm text-[var(--brand-text-secondary)]">This project uses its own SQLite file.</p>{/if}
      <div class="space-y-2 pb-3">
        {#if result !== undefined}<div class="alert alert-soft" class:alert-success={result.ok} class:alert-error={!result.ok} role={result.ok ? 'status' : 'alert'}>{result.message}</div>{/if}
        {#if message !== ''}<p class="text-xs" class:text-error={messageError} role={messageError ? 'alert' : 'status'}>{message}</p>{/if}
      </div>
    </div>
  {/if}
</SettingsSection>

<ConfirmModal open={confirmChoice !== undefined} title={confirmationTitle()} body={confirmChoice === 'sqlite' ? 'This removes the project connection. Data in the database server is kept.' : 'The current database connection will change. The external database and its data stay untouched.'} confirmLabel={confirmChoice === 'sqlite' ? 'Use SQLite' : 'Continue'} cancelLabel="Cancel" oncancel={() => { confirmChoice = undefined; }} onconfirm={confirmChange} />
