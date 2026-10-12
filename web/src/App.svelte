<script>
  import HelloCard from './lib/components/HelloCard.svelte';
  import { APP_DESCRIPTION, APP_TITLE } from './lib/constants/app.js';
</script>

<main class="hero min-h-screen bg-base-200">
  <div class="hero-content w-full text-center">
    <HelloCard title={APP_TITLE} message={APP_DESCRIPTION} />
  </div>
</main>
