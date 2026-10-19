<script lang="ts">
  import { integrationsCopy } from '../content/integrations';
  import GitHubIntegration from './GitHubIntegration.svelte';
  import SettingsSection from './parts/SettingsSection.svelte';
</script>

<SettingsSection title={integrationsCopy.title} description={integrationsCopy.intro}>
  <ul class="list mt-3 rounded-box border border-base-300 bg-base-100">
    <GitHubIntegration />
  </ul>
</SettingsSection>
