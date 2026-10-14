<script lang="ts">
  import { comingSoon } from '../content/models';
  import { navigate } from '../../../app/router';
  import { session, signOut } from '../../../shared/session/session';
  import ChangePasswordForm from './parts/ChangePasswordForm.svelte';
  import SettingRow from './parts/SettingRow.svelte';
  import SettingsSection from './parts/SettingsSection.svelte';

  const user = $derived($session.status === 'signed-in' ? $session.user : { name: '', email: '' });

  async function leave(): Promise<void> {
    await signOut();
    navigate('/login');
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
  <SettingRow label="Export my data" description={comingSoon.export}><span class="badge badge-ghost badge-sm">{comingSoon.label}</span></SettingRow>
  <SettingRow label="Delete all tasks" description={comingSoon.delete}><span class="badge badge-ghost badge-sm">{comingSoon.label}</span></SettingRow>
</SettingsSection>

<SettingsSection title="Session">
  <SettingRow label="Sign out" description="You'll need to sign in again on this device."><button class="btn btn-sm" type="button" onclick={leave}>Sign out</button></SettingRow>
</SettingsSection>
