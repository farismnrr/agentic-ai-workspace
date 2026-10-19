<script lang="ts">
  import { integrationsCopy } from '../content/integrations';
  import { subscriptionsCopy } from '../content/subscriptions';
  import GitHubIntegration from './GitHubIntegration.svelte';
  import SettingsSection from './parts/SettingsSection.svelte';
  import SubscriptionRow from './SubscriptionRow.svelte';
  let version = $state(0);
</script>

<SettingsSection title={integrationsCopy.title} description={integrationsCopy.intro}>
  <ul class="list mt-3 rounded-box border border-base-300 bg-base-100">
    <GitHubIntegration />
  </ul>
</SettingsSection>
<SettingsSection title={subscriptionsCopy.title} description={subscriptionsCopy.intro}>
  <ul class="list mt-3 rounded-box border border-base-300 bg-base-100">
    <SubscriptionRow program="claude" {version} onchange={() => (version += 1)} />
    <SubscriptionRow program="chatgpt" {version} onchange={() => (version += 1)} />
  </ul>
</SettingsSection>
