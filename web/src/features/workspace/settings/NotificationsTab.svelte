<script lang="ts">
  import { notificationsCopy } from '../content/models';
  import { askPermission, loadChoices, permission, saveChoices, supported } from '../logic/notify';
  import SettingRow from './parts/SettingRow.svelte';
  import SettingsSection from './parts/SettingsSection.svelte';

  let choices = $state(loadChoices());
  let allowed = $state(permission());

  const save = (): void => {
    saveChoices({ ...choices });
  };

  async function enable(): Promise<void> {
    allowed = await askPermission();
  }
</script>

<SettingsSection title="On this device" description={notificationsCopy.description}>
  {#if !supported()}
    <p class="py-3 text-xs text-[var(--brand-text-secondary)]">{notificationsCopy.unsupported}</p>
  {:else if allowed !== 'granted'}
    <SettingRow label="Allow notifications" description={allowed === 'denied' ? notificationsCopy.blocked : notificationsCopy.ask}>
      <button class="btn btn-sm btn-primary" type="button" disabled={allowed === 'denied'} onclick={enable}>Allow</button>
    </SettingRow>
  {:else}
    <SettingRow label="When it needs me" description={notificationsCopy.asks} for="notify-asks">
      <input id="notify-asks" type="checkbox" class="toggle toggle-primary toggle-sm" bind:checked={choices.asks} onchange={save} />
    </SettingRow>
    <SettingRow label="When something goes wrong" description={notificationsCopy.problems} for="notify-problems">
      <input id="notify-problems" type="checkbox" class="toggle toggle-primary toggle-sm" bind:checked={choices.problems} onchange={save} />
    </SettingRow>
  {/if}
</SettingsSection>
