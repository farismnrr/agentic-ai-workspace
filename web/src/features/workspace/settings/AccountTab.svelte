<script lang="ts">
  import { navigate } from '../../../app/router';
  import { session, signOut } from '../../../shared/session/session';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import { deleteAllTasks } from '../logic/account-api';
  import { refresh } from '../logic/workspace-store';
  import ChangePasswordForm from './parts/ChangePasswordForm.svelte';
  import DeleteAccountModal from './parts/DeleteAccountModal.svelte';
  import SettingRow from './parts/SettingRow.svelte';
  import SettingsSection from './parts/SettingsSection.svelte';

  const user = $derived($session.status === 'signed-in' ? $session.user : { name: '', email: '' });
  let confirmTasks = $state(false);
  let confirmAccount = $state(false);
  let failed = $state(false);

  async function leave(): Promise<void> {
    await signOut();
    navigate('/login');
  }

  async function clearTasks(): Promise<void> {
    confirmTasks = false;
    failed = false;
    try {
      await deleteAllTasks();
      await refresh();
    } catch {
      failed = true;
    }
  }

  async function accountDeleted(): Promise<void> {
    confirmAccount = false;
    await signOut();
    navigate('/');
  }
</script>

<SettingsSection title="Profile">
  <SettingRow label="Name"><span>{user.name}</span></SettingRow>
  <SettingRow label="Email"><span>{user.email}</span></SettingRow>
</SettingsSection>

<SettingsSection title="Password" description="Changing it signs you out on your other devices.">
  <ChangePasswordForm />
</SettingsSection>

<SettingsSection title="Your data" description="Everything your AI team made for you belongs to you.">
  <SettingRow label="Export my data" description="A zip with your settings, projects and their files, and every task with its steps. Saved AI keys are left out.">
    <a class="btn btn-sm" href="/api/account/export" download>Download</a>
  </SettingRow>
  <SettingRow label="Delete all tasks" description="Remove every task and its history. Your projects and files stay.">
    <button class="btn btn-sm" type="button" onclick={() => (confirmTasks = true)}>Delete tasks</button>
  </SettingRow>
  {#if failed}<p class="py-2 text-xs text-error" role="alert">Could not delete your tasks. Please try again.</p>{/if}
  <SettingRow label="Delete my account" description="Remove your account and everything in it for good.">
    <button class="btn btn-sm btn-error btn-outline" type="button" onclick={() => (confirmAccount = true)}>Delete account</button>
  </SettingRow>
</SettingsSection>

<SettingsSection title="Session">
  <SettingRow label="Sign out" description="You'll need to sign in again on this device."><button class="btn btn-sm" type="button" onclick={leave}>Sign out</button></SettingRow>
</SettingsSection>

<ConfirmModal
  open={confirmTasks}
  title="Delete all tasks?"
  body="Every task and its history is removed, including running ones. Projects and their files stay. This cannot be undone."
  confirmLabel="Delete tasks"
  cancelLabel="Keep them"
  oncancel={() => (confirmTasks = false)}
  onconfirm={clearTasks}
/>
<DeleteAccountModal open={confirmAccount} oncancel={() => (confirmAccount = false)} ondeleted={accountDeleted} />
