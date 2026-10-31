<script lang="ts">
  import { onMount } from 'svelte';
  import { personaCopy } from '../content/models';
  import { loadPreferences, savePreferences, type Preferences } from '../logic/preferences-api';
  import SettingRow from './parts/SettingRow.svelte';
  import SaveBar from './parts/SaveBar.svelte';
  import SettingsSection from './parts/SettingsSection.svelte';
  import RulesList from './RulesList.svelte';

  let prefs: Preferences | undefined = $state();
  let saving = $state(false);
  let message = $state('');
  let failed = $state(false);

  onMount(async () => {
    try {
      prefs = await loadPreferences();
    } catch {
      failed = true;
      message = personaCopy.failed;
    }
  });

  async function save(): Promise<void> {
    if (prefs === undefined) return;
    saving = true;
    try {
      prefs = await savePreferences(prefs);
      failed = false;
      message = personaCopy.saved;
    } catch {
      failed = true;
      message = personaCopy.failed;
    } finally {
      saving = false;
    }
  }
</script>

{#if prefs !== undefined}
  <SettingsSection title="Persona" description="How AI responds to you.">
    <SettingRow label="Name" for="persona-name"><input id="persona-name" class="input input-sm w-64" placeholder="Awam" bind:value={prefs.personaName} /></SettingRow>
    <SettingRow label="How it should talk" for="style">
      <select id="style" class="select select-sm w-64" bind:value={prefs.talkStyle}>
        <option value="">No preference</option>
        {#each personaCopy.talkStyles as option (option)}<option>{option}</option>{/each}
      </select>
    </SettingRow>
    <SettingRow label="Language" for="language">
      <select id="language" class="select select-sm w-64" bind:value={prefs.language}>
        <option value="">Match my message</option>
        {#each personaCopy.languages as option (option)}<option>{option}</option>{/each}
      </select>
    </SettingRow>
    <div class="py-3">
      <label class="font-medium" for="about">About you <span class="font-normal text-[var(--brand-text-tertiary)]">(optional)</span></label>
      <textarea id="about" class="textarea mt-1.5 w-full" rows="3" maxlength="1000" bind:value={prefs.about}></textarea>
    </div>
  </SettingsSection>

  <RulesList bind:rules={prefs.rules} />
  <SaveBar {saving} {message} {failed} onsave={save} />
{:else if failed}
  <p class="text-error" role="alert">{message}</p>
{:else}
  <span class="loading loading-spinner loading-sm" aria-label="Loading"></span>
{/if}
