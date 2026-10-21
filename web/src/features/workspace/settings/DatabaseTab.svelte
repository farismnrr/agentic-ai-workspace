<script lang="ts">
  import { onMount } from 'svelte';
  import { getSharedDatabaseSettings, saveSharedDatabaseSettings } from './database-api';

  const sslModes = ['disable', 'allow', 'prefer', 'require', 'verify-ca', 'verify-full'];
  let host = $state('');
  let port = $state(5432);
  let database = $state('');
  let user = $state('');
  let password = $state('');
  let sslMode = $state('prefer');
  let passwordSet = $state(false);
  let loading = $state(true);
  let saving = $state(false);
  let message = $state('');
  let failed = $state(false);

  onMount(async () => {
    try {
      const settings = await getSharedDatabaseSettings();
      host = settings.host;
      port = settings.port || 5432;
      database = settings.database;
      user = settings.user;
      sslMode = settings.sslMode || 'prefer';
      passwordSet = settings.passwordSet;
    } catch {
      failed = true;
      message = 'Could not load database settings.';
    } finally {
      loading = false;
    }
  });

  async function save(): Promise<void> {
    saving = true;
    failed = false;
    message = '';
    try {
      const result = await saveSharedDatabaseSettings({ host, port, database, user, password, sslMode });
      password = '';
      passwordSet = result.passwordSet;
      message = 'Database settings saved.';
    } catch {
      failed = true;
      message = 'Could not save these settings. Check the fields and try again.';
    } finally {
      saving = false;
    }
  }
</script>

<section class="space-y-5" aria-labelledby="database-settings-title">
  <div>
    <h2 id="database-settings-title" class="text-lg font-semibold">Shared PostgreSQL</h2>
    <p class="mt-1 text-sm text-base-content/70">Projects can use this account to create a private database schema.</p>
  </div>
  {#if loading}<p role="status" class="text-sm text-base-content/70">Loading database settings…</p>
  {:else}
    <form class="grid gap-4 sm:grid-cols-2" onsubmit={(event) => { event.preventDefault(); void save(); }}>
      <label class="form-control gap-1"><span class="label-text">Host</span><input class="input input-bordered w-full" bind:value={host} required /></label>
      <label class="form-control gap-1"><span class="label-text">Port</span><input class="input input-bordered w-full" type="number" min="1" max="65535" bind:value={port} required /></label>
      <label class="form-control gap-1"><span class="label-text">Database</span><input class="input input-bordered w-full" bind:value={database} required /></label>
      <label class="form-control gap-1"><span class="label-text">User</span><input class="input input-bordered w-full" bind:value={user} required /></label>
      <label class="form-control gap-1"><span class="label-text">Password</span><input class="input input-bordered w-full" type="password" bind:value={password} autocomplete="new-password" placeholder={passwordSet ? 'Saved; leave blank to keep it' : ''} required={!passwordSet} /></label>
      <label class="form-control gap-1"><span class="label-text">SSL mode</span><select class="select select-bordered w-full" bind:value={sslMode}>{#each sslModes as mode (mode)}<option value={mode}>{mode}</option>{/each}</select></label>
      <div class="flex flex-wrap items-center gap-3 sm:col-span-2">
        <button class="btn btn-primary" type="submit" disabled={saving}>{saving ? 'Saving…' : 'Save database settings'}</button>
        {#if message}<p class="text-sm {failed ? 'text-error' : 'text-success'}" role={failed ? 'alert' : 'status'}>{message}</p>{/if}
      </div>
    </form>
  {/if}
</section>
