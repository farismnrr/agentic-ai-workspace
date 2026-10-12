<script lang="ts">
  interface Props {
    title: string;
    message: string;
  }

  let { title, message }: Props = $props();
</script>

<section class="card w-full max-w-lg bg-base-100 shadow-xl">
  <div class="card-body items-center text-center">
    <div class="badge badge-primary badge-outline">Go + Svelte</div>
    <h1 class="card-title text-3xl">{title}</h1>
    <p class="text-base-content/70">{message}</p>
  </div>
</section>
