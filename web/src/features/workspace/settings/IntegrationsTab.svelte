<script lang="ts">
  import { integrationsCopy } from '../content/integrations';
  import GitHubIntegration from './GitHubIntegration.svelte';
  import SettingsSection from './parts/SettingsSection.svelte';
</script>

<SettingsSection title={integrationsCopy.title}>
  <ul class="list">
    <GitHubIntegration />
  </ul>
</SettingsSection>
