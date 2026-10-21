<script lang="ts">
  import type { DatabaseType } from '../env-api';

  interface Props {
    type: DatabaseType; host: string; port: number; database: string; user: string;
    password: string; tls: boolean; envName: string; configured: boolean;
    ontype: (value: DatabaseType) => void;
    onhost: (value: string) => void; onport: (value: number) => void;
    ondatabase: (value: string) => void; onuser: (value: string) => void;
    onpassword: (value: string) => void; ontls: (value: boolean) => void;
    onenvname: (value: string) => void;
  }
  const props: Props = $props();
</script>

<div class="grid gap-3 sm:grid-cols-2">
  <label>
    <span class="label-text mb-1 block">Database type</span>
    <select class="select select-sm w-full" value={props.type} onchange={(event) => { props.ontype(event.currentTarget.value as DatabaseType); }}>
      <option value="postgres">PostgreSQL</option><option value="mysql">MySQL / MariaDB</option><option value="mongodb">MongoDB</option>
    </select>
  </label>
  <label><span class="label-text mb-1 block">Host</span><input class="input input-sm w-full" value={props.host} oninput={(event) => { props.onhost(event.currentTarget.value); }} required autocomplete="off" placeholder="db.example.com" /></label>
  <label><span class="label-text mb-1 block">Port</span><input class="input input-sm w-full" type="number" value={props.port} oninput={(event) => { props.onport(Number(event.currentTarget.value)); }} min="1" max="65535" required /></label>
  <label><span class="label-text mb-1 block">Database</span><input class="input input-sm w-full" value={props.database} oninput={(event) => { props.ondatabase(event.currentTarget.value); }} required autocomplete="off" /></label>
  <label><span class="label-text mb-1 block">User</span><input class="input input-sm w-full" value={props.user} oninput={(event) => { props.onuser(event.currentTarget.value); }} required autocomplete="username" /></label>
  <label>
    <span class="label-text mb-1 block">Password</span>
    <input class="input input-sm w-full" type="password" value={props.password} oninput={(event) => { props.onpassword(event.currentTarget.value); }} autocomplete="new-password" placeholder={props.configured ? 'Enter a new password to save' : 'Database password'} />
    <span class="mt-1 block text-xs text-[var(--brand-text-secondary)]">{props.configured ? 'Password saved; enter a new one when saving changes.' : 'Write-only; never shown after saving.'}</span>
  </label>
  <label>
    <span class="label-text mb-1 block">Environment variable name</span>
    <input class="input input-sm w-full font-mono uppercase" value={props.envName} oninput={(event) => { props.onenvname(event.currentTarget.value.toUpperCase()); }} maxlength="64" pattern="[A-Za-z_][A-Za-z0-9_]*" required />
  </label>
</div>
<label class="flex w-fit items-center gap-2 text-sm"><input class="checkbox checkbox-sm" type="checkbox" checked={props.tls} onchange={(event) => { props.ontls(event.currentTarget.checked); }} />Use TLS</label>
