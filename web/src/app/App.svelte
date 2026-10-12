<script lang="ts">
  import { APP_DESCRIPTION, APP_TITLE } from '../shared/config/app';
  import HelloCard from '../shared/ui/HelloCard.svelte';
</script>

<main class="hero min-h-screen bg-base-200">
  <div class="hero-content w-full text-center">
    <HelloCard title={APP_TITLE} message={APP_DESCRIPTION} />
  </div>
</main>
